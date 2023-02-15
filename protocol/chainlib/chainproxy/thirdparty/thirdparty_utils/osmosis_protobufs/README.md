###Osmosis Compiled Protobuf Files
These compiled protobuf files are copied from the Osmosis v14 commit [d2772a1e732145e45f195137a6978235bd1a2005](https://github.com/osmosis-labs/osmosis/tree/d2772a1e732145e45f195137a6978235bd1a2005 "d2772a1e732145e45f195137a6978235bd1a2005").

To update do the following:
1. Delete everything in this directory except for this file.
2. Copy all the .pb.go and .pb.gw.go under all the folders in the x folder of Osmosis and copy it keeping the folder structure.
3. Replace "github.com/osmosis-labs/osmosis/v14/x" with "github.com/lavanet/lava/protocol/chainlib/chainproxy/thirdparty/thirdparty_utils/osmosis_protobufs"
4. Create the missing String() methods using this as a template
`func (m *MissingType) String() string { return proto.CompactTextString(m) }`
5. Update the commit hash on this file.
6. Update the files in the [thirdparty/Osmosis](https://github.com/lavanet/lava/tree/main/protocol/chainlib/chainproxy/thirdparty/Osmosis "thirdparty/Osmosis") folder to reflect the changes.