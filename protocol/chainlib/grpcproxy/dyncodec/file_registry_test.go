package dyncodec

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestFileDescriptorSetRegistry(t *testing.T) {
	t.Run("NewFileDescriptorSetRegistry with nil", func(t *testing.T) {
		_, err := NewFileDescriptorSetRegistry(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("NewFileDescriptorSetRegistry with empty set", func(t *testing.T) {
		fds := &descriptorpb.FileDescriptorSet{}
		reg, err := NewFileDescriptorSetRegistry(fds)
		require.NoError(t, err)
		require.NotNil(t, reg)

		assert.Equal(t, 0, reg.FileCount())
		assert.Equal(t, 0, reg.SymbolCount())

		err = reg.Close()
		assert.NoError(t, err)
	})

	t.Run("FileDescriptorSetRegistry with proto file", func(t *testing.T) {
		// Create a minimal FileDescriptorSet with a test proto
		testFile := &descriptorpb.FileDescriptorProto{
			Name:    strPtr("test.proto"),
			Package: strPtr("test.package"),
			MessageType: []*descriptorpb.DescriptorProto{
				{
					Name: strPtr("TestMessage"),
					Field: []*descriptorpb.FieldDescriptorProto{
						{
							Name:   strPtr("id"),
							Number: int32Ptr(1),
							Type:   descriptorpb.FieldDescriptorProto_TYPE_INT64.Enum(),
						},
					},
				},
			},
			Service: []*descriptorpb.ServiceDescriptorProto{
				{
					Name: strPtr("TestService"),
					Method: []*descriptorpb.MethodDescriptorProto{
						{
							Name:       strPtr("GetTest"),
							InputType:  strPtr(".test.package.TestMessage"),
							OutputType: strPtr(".test.package.TestMessage"),
						},
					},
				},
			},
		}

		fds := &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{testFile},
		}

		reg, err := NewFileDescriptorSetRegistry(fds)
		require.NoError(t, err)
		require.NotNil(t, reg)

		// Test file count
		assert.Equal(t, 1, reg.FileCount())

		// Test symbol count (message, service, method)
		assert.True(t, reg.SymbolCount() > 0)

		// Test ProtoFileByPath
		file, err := reg.ProtoFileByPath("test.proto")
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		// Test ProtoFileByPath with non-existent path
		_, err = reg.ProtoFileByPath("nonexistent.proto")
		require.Error(t, err)

		// Test ProtoFileContainingSymbol
		file, err = reg.ProtoFileContainingSymbol(protoreflect.FullName("test.package.TestMessage"))
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		// Test service symbol
		file, err = reg.ProtoFileContainingSymbol(protoreflect.FullName("test.package.TestService"))
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		// Test method symbol
		file, err = reg.ProtoFileContainingSymbol(protoreflect.FullName("test.package.TestService.GetTest"))
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		// Test HasSymbol
		assert.True(t, reg.HasSymbol("test.package.TestMessage"))
		assert.False(t, reg.HasSymbol("nonexistent.Symbol"))

		// Test ListFiles
		files := reg.ListFiles()
		assert.Contains(t, files, "test.proto")

		// Test Close
		err = reg.Close()
		assert.NoError(t, err)

		// After close, operations should fail
		_, err = reg.ProtoFileByPath("test.proto")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("Symbol lookup with leading dot", func(t *testing.T) {
		testFile := &descriptorpb.FileDescriptorProto{
			Name:    strPtr("test.proto"),
			Package: strPtr("test"),
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: strPtr("Msg")},
			},
		}

		fds := &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{testFile},
		}

		reg, err := NewFileDescriptorSetRegistry(fds)
		require.NoError(t, err)
		defer reg.Close()

		// Test with leading dot (some callers pass ".package.Name")
		file, err := reg.ProtoFileContainingSymbol(protoreflect.FullName(".test.Msg"))
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())
	})
}

func TestHybridProtoFileRegistry(t *testing.T) {
	t.Run("Create with no sources fails", func(t *testing.T) {
		_, err := NewHybridProtoFileRegistry(HybridRegistryConfig{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "at least one descriptor source")
	})

	t.Run("Create with file descriptor set only", func(t *testing.T) {
		testFile := &descriptorpb.FileDescriptorProto{
			Name:    strPtr("test.proto"),
			Package: strPtr("test"),
			MessageType: []*descriptorpb.DescriptorProto{
				{Name: strPtr("Msg")},
			},
		}

		fds := &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{testFile},
		}

		reg, err := NewHybridProtoFileRegistry(HybridRegistryConfig{
			FileDescriptorSet: fds,
		})
		require.NoError(t, err)
		require.NotNil(t, reg)

		// Reflection should be marked unavailable
		assert.False(t, reg.IsReflectionAvailable())
		assert.True(t, reg.HasFileFallback())

		// Should use file registry
		file, err := reg.ProtoFileByPath("test.proto")
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		file, err = reg.ProtoFileContainingSymbol(protoreflect.FullName("test.Msg"))
		require.NoError(t, err)
		assert.Equal(t, "test.proto", file.GetName())

		err = reg.Close()
		assert.NoError(t, err)

		// After close, should fail
		_, err = reg.ProtoFileByPath("test.proto")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "closed")
	})

	t.Run("ResetReflectionStatus", func(t *testing.T) {
		testFile := &descriptorpb.FileDescriptorProto{
			Name:    strPtr("test.proto"),
			Package: strPtr("test"),
		}

		fds := &descriptorpb.FileDescriptorSet{
			File: []*descriptorpb.FileDescriptorProto{testFile},
		}

		reg, err := NewHybridProtoFileRegistry(HybridRegistryConfig{
			FileDescriptorSet: fds,
		})
		require.NoError(t, err)
		defer reg.Close()

		// Initially no reflection (file only)
		assert.False(t, reg.IsReflectionAvailable())

		// Reset should work (though still unavailable without connection)
		reg.ResetReflectionStatus()
		assert.False(t, reg.IsReflectionAvailable())
	})
}

// Helper functions for creating pointers
func strPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}
