package grpcutil

import (
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
)

type DescriptorSource interface {
	// ListServices returns a list of fully-qualified service names. It will be all services in a set of
	// descriptor files or the set of all services exposed by a gRPC server.
	ListServices() ([]string, error)
	// FindSymbol returns a descriptor for the given fully-qualified symbol name.
	FindSymbol(fullyQualifiedName string) (desc.Descriptor, error)
	// AllExtensionsForType returns all known extension fields that extend the given message type name.
	AllExtensionsForType(typeName string) ([]*desc.FieldDescriptor, error)
}

type serverSource struct {
	client *grpcreflect.Client
}

func (ss serverSource) ListServices() ([]string, error) {
	return ss.client.ListServices()
}

func (ss serverSource) FindSymbol(fullyQualifiedName string) (desc.Descriptor, error) {
	file, err := ss.client.FileContainingSymbol(fullyQualifiedName)
	if err != nil {
		return nil, errors.Wrap(err, "ss.client.FileContainingSymbol()")
	}

	d := file.FindSymbol(fullyQualifiedName)
	if d == nil {
		return nil, errors.New("must not be nil")
	}

	return d, nil
}

func (ss serverSource) AllExtensionsForType(typeName string) (fieldDescriptorSlice []*desc.FieldDescriptor, err error) {
	var nSlice []int32
	if nSlice, err = ss.client.AllExtensionNumbersForType(typeName); err != nil {
		return nil, errors.Wrap(err, "ss.client.AllExtensionNumbersForType()")
	}

	fieldDescriptorSlice = make([]*desc.FieldDescriptor, 0, len(nSlice))
	for _, n := range nSlice {
		var fieldDescriptor *desc.FieldDescriptor
		if fieldDescriptor, err = ss.client.ResolveExtension(typeName, n); err != nil {
			return nil, errors.Wrap(err, "ss.client.ResolveExtension()")
		}

		fieldDescriptorSlice = append(fieldDescriptorSlice, fieldDescriptor)
	}

	return fieldDescriptorSlice, nil
}

func descriptorSourceFromServer(refClient *grpcreflect.Client) DescriptorSource {
	return serverSource{client: refClient}
}
