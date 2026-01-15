package dyncodec

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/lavanet/lava/v5/utils"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// FileDescriptorSetRegistry implements ProtoFileRegistry by loading descriptors
// from a pre-compiled FileDescriptorSet file (.pb format).
//
// This is used when gRPC server reflection is disabled or unavailable.
// The FileDescriptorSet can be generated using protoc:
//
//	protoc --descriptor_set_out=output.pb --include_imports *.proto
//
// Or using buf:
//
//	buf build -o output.pb
type FileDescriptorSetRegistry struct {
	// filesByPath maps file path -> FileDescriptorProto
	filesByPath map[string]*descriptorpb.FileDescriptorProto

	// symbolIndex maps symbol full name -> file path (for fast lookup)
	symbolIndex map[string]string

	// mutex protects the maps
	mu sync.RWMutex

	// closed indicates if the registry has been closed
	closed bool
}

// NewFileDescriptorSetRegistry creates a new registry from a FileDescriptorSet.
// The FileDescriptorSet should contain all necessary proto files and their dependencies.
func NewFileDescriptorSetRegistry(fds *descriptorpb.FileDescriptorSet) (*FileDescriptorSetRegistry, error) {
	if fds == nil {
		return nil, fmt.Errorf("FileDescriptorSet is nil")
	}

	registry := &FileDescriptorSetRegistry{
		filesByPath: make(map[string]*descriptorpb.FileDescriptorProto),
		symbolIndex: make(map[string]string),
	}

	// Index all files
	for _, file := range fds.GetFile() {
		if file.GetName() == "" {
			continue
		}
		registry.filesByPath[file.GetName()] = file

		// Index all symbols in this file
		registry.indexFileSymbols(file)
	}

	utils.LavaFormatDebug("FileDescriptorSetRegistry initialized",
		utils.LogAttr("files", len(registry.filesByPath)),
		utils.LogAttr("symbols", len(registry.symbolIndex)))

	return registry, nil
}

// NewFileDescriptorSetRegistryFromPath loads a FileDescriptorSet from a file path.
// Supports binary .pb format.
func NewFileDescriptorSetRegistryFromPath(path string) (*FileDescriptorSetRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, utils.LavaFormatError("failed to read FileDescriptorSet file", err,
			utils.LogAttr("path", path))
	}

	fds := &descriptorpb.FileDescriptorSet{}
	if err := proto.Unmarshal(data, fds); err != nil {
		return nil, utils.LavaFormatError("failed to parse FileDescriptorSet", err,
			utils.LogAttr("path", path))
	}

	return NewFileDescriptorSetRegistry(fds)
}

// indexFileSymbols indexes all symbols (messages, enums, services, extensions) in a file
func (f *FileDescriptorSetRegistry) indexFileSymbols(file *descriptorpb.FileDescriptorProto) {
	pkg := file.GetPackage()
	filePath := file.GetName()

	// Index messages
	for _, msg := range file.GetMessageType() {
		f.indexMessage(pkg, filePath, msg)
	}

	// Index enums
	for _, enum := range file.GetEnumType() {
		fullName := joinNames(pkg, enum.GetName())
		f.symbolIndex[fullName] = filePath
	}

	// Index services
	for _, service := range file.GetService() {
		fullName := joinNames(pkg, service.GetName())
		f.symbolIndex[fullName] = filePath

		// Index service methods
		for _, method := range service.GetMethod() {
			methodFullName := fullName + "." + method.GetName()
			f.symbolIndex[methodFullName] = filePath
		}
	}

	// Index extensions
	for _, ext := range file.GetExtension() {
		fullName := joinNames(pkg, ext.GetName())
		f.symbolIndex[fullName] = filePath
	}
}

// indexMessage recursively indexes a message and its nested types
func (f *FileDescriptorSetRegistry) indexMessage(parentPkg, filePath string, msg *descriptorpb.DescriptorProto) {
	fullName := joinNames(parentPkg, msg.GetName())
	f.symbolIndex[fullName] = filePath

	// Index nested messages
	for _, nested := range msg.GetNestedType() {
		f.indexMessage(fullName, filePath, nested)
	}

	// Index nested enums
	for _, enum := range msg.GetEnumType() {
		enumFullName := joinNames(fullName, enum.GetName())
		f.symbolIndex[enumFullName] = filePath
	}

	// Index nested extensions
	for _, ext := range msg.GetExtension() {
		extFullName := joinNames(fullName, ext.GetName())
		f.symbolIndex[extFullName] = filePath
	}
}

// ProtoFileByPath returns the FileDescriptorProto for the given file path.
// Implements ProtoFileRegistry interface.
func (f *FileDescriptorSetRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return nil, fmt.Errorf("registry is closed")
	}

	file, ok := f.filesByPath[path]
	if !ok {
		return nil, fmt.Errorf("proto file not found: %s", path)
	}

	return file, nil
}

// ProtoFileContainingSymbol returns the FileDescriptorProto containing the given symbol.
// The symbol can be a message, enum, service, or method full name.
// Implements ProtoFileRegistry interface.
func (f *FileDescriptorSetRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.closed {
		return nil, fmt.Errorf("registry is closed")
	}

	symbolName := string(name)

	// Direct lookup
	if filePath, ok := f.symbolIndex[symbolName]; ok {
		if file, ok := f.filesByPath[filePath]; ok {
			return file, nil
		}
	}

	// Try without leading dot (some callers may pass ".package.Name")
	if strings.HasPrefix(symbolName, ".") {
		symbolName = symbolName[1:]
		if filePath, ok := f.symbolIndex[symbolName]; ok {
			if file, ok := f.filesByPath[filePath]; ok {
				return file, nil
			}
		}
	}

	// Try to find parent (for nested types or methods)
	// e.g., "cosmos.bank.v1beta1.Query.Balance" -> "cosmos.bank.v1beta1.Query"
	for symbolName != "" {
		lastDot := strings.LastIndex(symbolName, ".")
		if lastDot < 0 {
			break
		}
		symbolName = symbolName[:lastDot]
		if filePath, ok := f.symbolIndex[symbolName]; ok {
			if file, ok := f.filesByPath[filePath]; ok {
				return file, nil
			}
		}
	}

	return nil, fmt.Errorf("symbol not found: %s", name)
}

// Close closes the registry and releases resources.
// Implements ProtoFileRegistry interface.
func (f *FileDescriptorSetRegistry) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true
	f.filesByPath = nil
	f.symbolIndex = nil

	return nil
}

// FileCount returns the number of files in the registry
func (f *FileDescriptorSetRegistry) FileCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.filesByPath)
}

// SymbolCount returns the number of indexed symbols
func (f *FileDescriptorSetRegistry) SymbolCount() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.symbolIndex)
}

// HasSymbol checks if a symbol is indexed
func (f *FileDescriptorSetRegistry) HasSymbol(name string) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	_, ok := f.symbolIndex[name]
	return ok
}

// ListFiles returns all indexed file paths
func (f *FileDescriptorSetRegistry) ListFiles() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	files := make([]string, 0, len(f.filesByPath))
	for path := range f.filesByPath {
		files = append(files, path)
	}
	return files
}

// joinNames joins a package/parent name with a type name
func joinNames(parent, name string) string {
	if parent == "" {
		return name
	}
	return parent + "." + name
}

// Ensure FileDescriptorSetRegistry implements ProtoFileRegistry
var _ ProtoFileRegistry = (*FileDescriptorSetRegistry)(nil)
