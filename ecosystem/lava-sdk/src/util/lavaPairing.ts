import fs from "fs";

export async function fetchLavaPairing(path: string): Promise<any> {
  if (typeof window === "undefined") {
    // Running on the server
    const configFile = fs.readFileSync(path, "utf-8");
    return JSON.parse(configFile);
  } else {
    // Running in the browser
    const response = await fetch(path);
    return await response.json();
  }
}
