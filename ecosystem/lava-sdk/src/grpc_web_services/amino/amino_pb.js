/**
 * @fileoverview
 * @enhanceable
 * @suppress {messageConventions} JS Compiler reports an error if a variable or
 *     field starts with 'MSG_' and isn't a translatable message.
 * @public
 */
// GENERATED CODE -- DO NOT EDIT!

var jspb = require('google-protobuf');
var goog = jspb;
var global = Function('return this')();

var google_protobuf_descriptor_pb = require('google-protobuf/google/protobuf/descriptor_pb.js');
goog.exportSymbol('proto.amino.dontOmitempty', null, global);
goog.exportSymbol('proto.amino.encoding', null, global);
goog.exportSymbol('proto.amino.fieldName', null, global);
goog.exportSymbol('proto.amino.messageEncoding', null, global);
goog.exportSymbol('proto.amino.name', null, global);

/**
 * A tuple of {field number, class constructor} for the extension
 * field named `name`.
 * @type {!jspb.ExtensionFieldInfo<string>}
 */
proto.amino.name = new jspb.ExtensionFieldInfo(
    11110001,
    {name: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.MessageOptions.extensionsBinary[11110001] = new jspb.ExtensionFieldBinaryInfo(
    proto.amino.name,
    jspb.BinaryReader.prototype.readString,
    jspb.BinaryWriter.prototype.writeString,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.MessageOptions.extensions[11110001] = proto.amino.name;


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `messageEncoding`.
 * @type {!jspb.ExtensionFieldInfo<string>}
 */
proto.amino.messageEncoding = new jspb.ExtensionFieldInfo(
    11110002,
    {messageEncoding: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.MessageOptions.extensionsBinary[11110002] = new jspb.ExtensionFieldBinaryInfo(
    proto.amino.messageEncoding,
    jspb.BinaryReader.prototype.readString,
    jspb.BinaryWriter.prototype.writeString,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.MessageOptions.extensions[11110002] = proto.amino.messageEncoding;


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `encoding`.
 * @type {!jspb.ExtensionFieldInfo<string>}
 */
proto.amino.encoding = new jspb.ExtensionFieldInfo(
    11110003,
    {encoding: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.FieldOptions.extensionsBinary[11110003] = new jspb.ExtensionFieldBinaryInfo(
    proto.amino.encoding,
    jspb.BinaryReader.prototype.readString,
    jspb.BinaryWriter.prototype.writeString,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.FieldOptions.extensions[11110003] = proto.amino.encoding;


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `fieldName`.
 * @type {!jspb.ExtensionFieldInfo<string>}
 */
proto.amino.fieldName = new jspb.ExtensionFieldInfo(
    11110004,
    {fieldName: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.FieldOptions.extensionsBinary[11110004] = new jspb.ExtensionFieldBinaryInfo(
    proto.amino.fieldName,
    jspb.BinaryReader.prototype.readString,
    jspb.BinaryWriter.prototype.writeString,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.FieldOptions.extensions[11110004] = proto.amino.fieldName;


/**
 * A tuple of {field number, class constructor} for the extension
 * field named `dontOmitempty`.
 * @type {!jspb.ExtensionFieldInfo<boolean>}
 */
proto.amino.dontOmitempty = new jspb.ExtensionFieldInfo(
    11110005,
    {dontOmitempty: 0},
    null,
     /** @type {?function((boolean|undefined),!jspb.Message=): !Object} */ (
         null),
    0);

google_protobuf_descriptor_pb.FieldOptions.extensionsBinary[11110005] = new jspb.ExtensionFieldBinaryInfo(
    proto.amino.dontOmitempty,
    jspb.BinaryReader.prototype.readBool,
    jspb.BinaryWriter.prototype.writeBool,
    undefined,
    undefined,
    false);
// This registers the extension field with the extended class, so that
// toObject() will function correctly.
google_protobuf_descriptor_pb.FieldOptions.extensions[11110005] = proto.amino.dontOmitempty;

goog.object.extend(exports, proto.amino);
