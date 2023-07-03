class WalletErrors {
  static errWalletNotInitialized: Error = new Error(
    "Wallet was not initialized"
  );
  static errZeroAccountDoesNotExists: Error = new Error(
    "Zero account does not exists in wallet"
  );
  static errInvalidPrivateKey: Error = new Error("Invalid private key");
}

export default WalletErrors;
