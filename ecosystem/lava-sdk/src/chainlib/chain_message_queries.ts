import { BaseChainMessageContainer } from "./chain_message";
import { CONSISTENCY_SELECT_ALLPROVIDERS } from "../common/common";

export function ShouldSendToAllProviders(
  chainMessage: BaseChainMessageContainer
): boolean {
  return (
    chainMessage.getApi().getCategory()?.getStateful() ==
    CONSISTENCY_SELECT_ALLPROVIDERS
  );
}

export function GetAddon(chainMessage: BaseChainMessageContainer): string {
  const addon = chainMessage.getApiCollection().getCollectionData()?.getAddOn();
  if (addon == undefined) {
    return "";
  }
  return addon;
}

export function IsSubscription(
  chainMessage: BaseChainMessageContainer
): boolean {
  const subs = chainMessage.getApi().getCategory()?.getSubscription();
  if (subs == undefined) {
    return false;
  }
  return subs;
}

export function IsHangingApi(chainMessage: BaseChainMessageContainer): boolean {
  const hanging = chainMessage.getApi().getCategory()?.getHangingApi();
  if (hanging == undefined) {
    return false;
  }
  return hanging;
}

export function GetComputeUnits(
  chainMessage: BaseChainMessageContainer
): number {
  const cu = chainMessage.getApi().getComputeUnits();
  if (cu == undefined) return 0;
  return cu;
}

export function GetStateful(chainMessage: BaseChainMessageContainer): number {
  const stateful = chainMessage.getApi().getCategory()?.getStateful();
  if (stateful == undefined) {
    return 0;
  }
  return stateful;
}
