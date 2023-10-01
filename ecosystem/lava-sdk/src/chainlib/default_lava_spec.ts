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
  spec.setAverageBlockTime(30000);
  spec.setAllowedBlockLagForQosSync(2);
  spec.setBlockDistanceForFinalizedData(0);
  spec.setBlocksInFinalizationProof(1);

  spec.setApiCollectionsList(apiCollectionList);

  return spec;
}
