type RouterKey = string;

const SEPARATOR = "|";

export function newRouterKey(extensions: string[]): RouterKey {
  const uniqueExtensions = new Set(extensions);
  return Array.from(uniqueExtensions).sort().join(SEPARATOR);
}
