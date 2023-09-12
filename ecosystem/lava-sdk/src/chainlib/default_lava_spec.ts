import {
  Api,
  ApiCollection,
  CollectionData,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";
import { Spec } from "../grpc_web_services/lavanet/lava/spec/spec_pb";

export function getDefaultLavaSpec(): Spec {
  const api = new Api();
  api.setEnabled(true);
  api.setName("abci_query");
  api.setComputeUnits(10);

  const apis: Array<Api> = [];
  apis.push(api);

  const collectionData = new CollectionData();
  collectionData.setApiInterface("tendermintrpc");

  const apiCollection = new ApiCollection();
  apiCollection.setEnabled(true);
  apiCollection.setCollectionData(collectionData);
  apiCollection.setApisList(apis);

  const apiCollectionList: Array<ApiCollection> = [];
  apiCollectionList.push(apiCollection);

  const spec = new Spec();
  spec.setEnabled(true);
  spec.setIndex("LAV1");
  // All of these will panic if we do not set them

  // TODO missing AverageBlockTime
  spec.setAverageBlockTime(123);

  // TODO misisng AllowedBlockLagForQosSync
  spec.setAllowedBlockLagForQosSync(12313);

  // TODO missing BlockDistanceForFinalizedData
  spec.setBlockDistanceForFinalizedData(12313);

  // todo missing BlocksInFinalizationProof
  spec.setBlocksInFinalizationProof(123123);

  // ----------------

  spec.setApiCollectionsList(apiCollectionList);

  return spec;
}
