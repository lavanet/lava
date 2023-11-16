import {
  Api,
  SpecCategory,
} from "../grpc_web_services/lavanet/lava/spec/api_collection_pb";

export function CombineSpecCategories(
  first: SpecCategory,
  second: SpecCategory
): SpecCategory {
  const combined = new SpecCategory();
  const firstObj = first.toObject();
  const secondObj = second.toObject();

  combined.setDeterministic(firstObj.deterministic && secondObj.deterministic);
  combined.setLocal(firstObj.local || secondObj.local);
  combined.setSubscription(firstObj.subscription || secondObj.subscription);
  combined.setStateful(firstObj.stateful + secondObj.stateful);
  combined.setHangingApi(firstObj.hangingApi || secondObj.hangingApi);
  return combined;
}
