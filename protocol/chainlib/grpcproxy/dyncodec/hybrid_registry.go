package dyncodec

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lavanet/lava/v5/utils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// HybridProtoFileRegistry implements ProtoFileRegistry by trying multiple sources.
// It first attempts gRPC reflection, and falls back to a file-based registry if reflection fails.
//
// This is useful for endpoints that may or may not have reflection enabled:
//   - If reflection works, use it (auto-discovery of new methods)
//   - If reflection fails, fall back to pre-compiled descriptors
//
// The registry caches the reflection availability status to avoid repeated probe attempts.
type HybridProtoFileRegistry struct {
	// Primary: gRPC reflection registry
	reflectionConn    *grpc.ClientConn
	reflectionTimeout time.Duration
	reflectionMu      sync.RWMutex
	reflectionReg     *GRPCReflectionProtoFileRegistry

	// Fallback: file-based registry
	fileReg *FileDescriptorSetRegistry

	// reflectionAvailable tracks if reflection is working
	// 0 = unknown, 1 = available, 2 = unavailable
	reflectionStatus atomic.Int32

	// closed indicates if the registry has been closed
	closed atomic.Bool
}

const (
	reflectionStatusUnknown     int32 = 0
	reflectionStatusAvailable   int32 = 1
	reflectionStatusUnavailable int32 = 2
)

// HybridRegistryConfig holds configuration for creating a HybridProtoFileRegistry
type HybridRegistryConfig struct {
	// GRPCConn is the gRPC connection for reflection queries
	GRPCConn *grpc.ClientConn

	// ReflectionTimeout is the timeout for reflection queries
	// Default: 5 seconds
	ReflectionTimeout time.Duration

	// FileDescriptorSet is the fallback descriptor set
	// Can be nil if reflection is expected to always work
	FileDescriptorSet *descriptorpb.FileDescriptorSet

	// FileDescriptorPath is an alternative to FileDescriptorSet - path to .pb file
	FileDescriptorPath string
}

// NewHybridProtoFileRegistry creates a new hybrid registry.
// At least one of grpcConn or fileDescriptorSet/path must be provided.
func NewHybridProtoFileRegistry(config HybridRegistryConfig) (*HybridProtoFileRegistry, error) {
	if config.GRPCConn == nil && config.FileDescriptorSet == nil && config.FileDescriptorPath == "" {
		return nil, fmt.Errorf("at least one descriptor source (gRPC connection or file) must be provided")
	}

	registry := &HybridProtoFileRegistry{
		reflectionConn:    config.GRPCConn,
		reflectionTimeout: config.ReflectionTimeout,
	}

	if registry.reflectionTimeout <= 0 {
		registry.reflectionTimeout = 5 * time.Second
	}

	// Initialize reflection registry if connection provided
	if config.GRPCConn != nil {
		registry.reflectionReg = NewGRPCReflectionProtoFileRegistryFromConn(config.GRPCConn)
	}

	// Initialize file registry
	var err error
	if config.FileDescriptorSet != nil {
		registry.fileReg, err = NewFileDescriptorSetRegistry(config.FileDescriptorSet)
		if err != nil {
			return nil, utils.LavaFormatError("failed to initialize file descriptor registry", err)
		}
	} else if config.FileDescriptorPath != "" {
		registry.fileReg, err = NewFileDescriptorSetRegistryFromPath(config.FileDescriptorPath)
		if err != nil {
			return nil, utils.LavaFormatError("failed to load file descriptor set", err,
				utils.LogAttr("path", config.FileDescriptorPath))
		}
	}

	// Determine initial status
	if config.GRPCConn == nil {
		// No reflection connection, mark as unavailable
		registry.reflectionStatus.Store(reflectionStatusUnavailable)
	}

	return registry, nil
}

// ProtoFileByPath returns the FileDescriptorProto for the given file path.
// Tries reflection first, falls back to file registry.
func (h *HybridProtoFileRegistry) ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	if h.closed.Load() {
		return nil, fmt.Errorf("registry is closed")
	}

	// Try reflection if available
	if h.shouldTryReflection() {
		file, err := h.tryReflectionByPath(path)
		if err == nil {
			return file, nil
		}
		// Mark reflection as unavailable if it fails
		h.markReflectionUnavailable(err)
	}

	// Fall back to file registry
	if h.fileReg != nil {
		return h.fileReg.ProtoFileByPath(path)
	}

	return nil, fmt.Errorf("proto file not found and no fallback available: %s", path)
}

// ProtoFileContainingSymbol returns the FileDescriptorProto containing the given symbol.
// Tries reflection first, falls back to file registry.
func (h *HybridProtoFileRegistry) ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	if h.closed.Load() {
		return nil, fmt.Errorf("registry is closed")
	}

	// Try reflection if available
	if h.shouldTryReflection() {
		file, err := h.tryReflectionBySymbol(name)
		if err == nil {
			return file, nil
		}
		// Mark reflection as unavailable if it fails
		h.markReflectionUnavailable(err)
	}

	// Fall back to file registry
	if h.fileReg != nil {
		return h.fileReg.ProtoFileContainingSymbol(name)
	}

	return nil, fmt.Errorf("symbol not found and no fallback available: %s", name)
}

// Close closes the registry and releases resources.
func (h *HybridProtoFileRegistry) Close() error {
	if !h.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	var errs []error

	if h.reflectionReg != nil {
		if err := h.reflectionReg.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if h.fileReg != nil {
		if err := h.fileReg.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing hybrid registry: %v", errs)
	}
	return nil
}

// shouldTryReflection returns true if we should attempt a reflection query
func (h *HybridProtoFileRegistry) shouldTryReflection() bool {
	if h.reflectionReg == nil {
		return false
	}

	status := h.reflectionStatus.Load()
	return status == reflectionStatusUnknown || status == reflectionStatusAvailable
}

// tryReflectionByPath attempts a reflection query by file path with timeout
func (h *HybridProtoFileRegistry) tryReflectionByPath(path string) (*descriptorpb.FileDescriptorProto, error) {
	h.reflectionMu.RLock()
	reg := h.reflectionReg
	h.reflectionMu.RUnlock()

	if reg == nil {
		return nil, fmt.Errorf("reflection registry not initialized")
	}

	// Use context with timeout for the reflection query
	ctx, cancel := context.WithTimeout(context.Background(), h.reflectionTimeout)
	defer cancel()

	// Run the query in a goroutine to respect timeout
	type result struct {
		file *descriptorpb.FileDescriptorProto
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		file, err := reg.ProtoFileByPath(path)
		resultCh <- result{file, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("reflection query timed out after %v", h.reflectionTimeout)
	case r := <-resultCh:
		if r.err == nil {
			// Reflection worked, mark as available
			h.reflectionStatus.Store(reflectionStatusAvailable)
		}
		return r.file, r.err
	}
}

// tryReflectionBySymbol attempts a reflection query by symbol with timeout
func (h *HybridProtoFileRegistry) tryReflectionBySymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error) {
	h.reflectionMu.RLock()
	reg := h.reflectionReg
	h.reflectionMu.RUnlock()

	if reg == nil {
		return nil, fmt.Errorf("reflection registry not initialized")
	}

	// Use context with timeout for the reflection query
	ctx, cancel := context.WithTimeout(context.Background(), h.reflectionTimeout)
	defer cancel()

	// Run the query in a goroutine to respect timeout
	type result struct {
		file *descriptorpb.FileDescriptorProto
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		file, err := reg.ProtoFileContainingSymbol(name)
		resultCh <- result{file, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("reflection query timed out after %v", h.reflectionTimeout)
	case r := <-resultCh:
		if r.err == nil {
			// Reflection worked, mark as available
			h.reflectionStatus.Store(reflectionStatusAvailable)
		}
		return r.file, r.err
	}
}

// markReflectionUnavailable marks reflection as unavailable and logs the reason
func (h *HybridProtoFileRegistry) markReflectionUnavailable(err error) {
	// Only log once when transitioning from unknown/available to unavailable
	if h.reflectionStatus.Swap(reflectionStatusUnavailable) != reflectionStatusUnavailable {
		if h.fileReg != nil {
			utils.LavaFormatInfo("gRPC reflection unavailable, using file-based descriptors",
				utils.LogAttr("reason", err.Error()))
		} else {
			utils.LavaFormatWarning("gRPC reflection unavailable and no fallback configured",
				err)
		}
	}
}

// IsReflectionAvailable returns true if reflection is currently working
func (h *HybridProtoFileRegistry) IsReflectionAvailable() bool {
	return h.reflectionStatus.Load() == reflectionStatusAvailable
}

// HasFileFallback returns true if a file-based fallback is configured
func (h *HybridProtoFileRegistry) HasFileFallback() bool {
	return h.fileReg != nil
}

// ResetReflectionStatus resets the reflection status to unknown,
// allowing the next query to probe reflection again.
// Useful after reconnection or endpoint changes.
func (h *HybridProtoFileRegistry) ResetReflectionStatus() {
	h.reflectionStatus.Store(reflectionStatusUnknown)
}

// Ensure HybridProtoFileRegistry implements ProtoFileRegistry
var _ ProtoFileRegistry = (*HybridProtoFileRegistry)(nil)
