package dyncodec

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lavanet/lava/v2/utils"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func NewRegistry(remote ProtoFileRegistry) *Registry {
	return &Registry{
		remote:    remote,
		prefFiles: new(protoregistry.Files),
		prefTypes: new(protoregistry.Types),
	}
}

var (
	_ protodesc.Resolver                  = (*Registry)(nil)
	_ protoregistry.ExtensionTypeResolver = (*Registry)(nil)
	_ protoregistry.MessageTypeResolver   = (*Registry)(nil)
)

type ProtoFileRegistry interface {
	ProtoFileByPath(path string) (*descriptorpb.FileDescriptorProto, error)
	ProtoFileContainingSymbol(name protoreflect.FullName) (*descriptorpb.FileDescriptorProto, error)
	Close() error
}

type Registry struct {
	// TODO(fdymylja): probably needs to be protected by a mutex.
	remote ProtoFileRegistry

	prefFiles *protoregistry.Files
	prefTypes *protoregistry.Types
}

func (r *Registry) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	// try in types
	xt, err := r.prefTypes.FindExtensionByName(field)
	if err == nil {
		return xt, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}
	// not found try in files
	xd, err := r.FindDescriptorByName(field)
	if err != nil {
		return nil, err
	}

	xdExtensionDescriptor, ok := xd.(protoreflect.ExtensionDescriptor)
	if !ok {
		return nil, utils.LavaFormatError("Failed converting xd.(protoreflect.ExtensionDescriptor)", nil, utils.Attribute{Key: "xd", Value: xd})
	}

	xt = dynamicpb.NewExtensionType(xdExtensionDescriptor)
	return xt, r.prefTypes.RegisterExtension(xt)
}

func (r *Registry) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, fmt.Errorf("not supported")
}

func (r *Registry) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	mt, err := r.prefTypes.FindMessageByName(message)
	if err == nil {
		return mt, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	md, err := r.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}
	messageDescriptor, ok := md.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, utils.LavaFormatError("Failed converting md.(protoreflect.MessageDescriptor)", nil, utils.Attribute{Key: "md", Value: md})
	}

	mt = dynamicpb.NewMessageType(messageDescriptor)
	return mt, r.prefTypes.RegisterMessage(mt)
}

func (r *Registry) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := r.prefTypes.FindMessageByURL(url)
	if err == nil {
		return mt, err
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	message := fullNameFromURL(url)

	md, err := r.FindDescriptorByName(message)
	if err != nil {
		return nil, err
	}

	messageDescriptor, ok := md.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, utils.LavaFormatError("Failed converting md.(protoreflect.MessageDescriptor)", nil, utils.Attribute{Key: "md", Value: md})
	}

	mt = dynamicpb.NewMessageType(messageDescriptor)
	return mt, r.prefTypes.RegisterMessage(mt)
}

func (r *Registry) FindFileByPath(s string) (protoreflect.FileDescriptor, error) {
	fd, err := r.prefFiles.FindFileByPath(s)
	if err == nil {
		return fd, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	dpb, err := r.remote.ProtoFileByPath(s)
	if err != nil {
		return nil, err
	}

	fd, err = protodesc.FileOptions{
		AllowUnresolvable: true,
	}.New(dpb, r)
	if err != nil {
		return nil, err
	}

	err = r.prefFiles.RegisterFile(fd)
	if err != nil {
		return nil, err
	}
	return fd, nil
}

func (r *Registry) FindDescriptorByName(name protoreflect.FullName) (protoreflect.Descriptor, error) {
	desc, err := r.prefFiles.FindDescriptorByName(name)
	if err == nil {
		return desc, nil
	}
	if !errors.Is(err, protoregistry.NotFound) {
		return nil, err
	}

	dpb, err := r.remote.ProtoFileContainingSymbol(name)
	if err != nil {
		return nil, err
	}
	fd, err := protodesc.FileOptions{
		AllowUnresolvable: true,
	}.New(dpb, r)
	if err != nil {
		return nil, err
	}

	err = r.prefFiles.RegisterFile(fd)
	if err != nil {
		return nil, err
	}

	return r.prefFiles.FindDescriptorByName(name)
}

func (r *Registry) Save() (*descriptorpb.FileDescriptorSet, error) {
	set := &descriptorpb.FileDescriptorSet{File: make([]*descriptorpb.FileDescriptorProto, 0, r.prefFiles.NumFiles())}
	var err error
	r.prefFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		set.File = append(set.File, protodesc.ToFileDescriptorProto(fd))
		return true
	})

	return set, err
}

func (r *Registry) Remote() ProtoFileRegistry {
	return r.remote
}

// fullNameFromURL returns protoreflect.FullName from proto.Messages' typeURL
func fullNameFromURL(typeURL string) protoreflect.FullName {
	message := protoreflect.FullName(typeURL)
	if i := strings.LastIndexByte(typeURL, '/'); i >= 0 {
		message = message[i+len("/"):]
	}

	return message
}
