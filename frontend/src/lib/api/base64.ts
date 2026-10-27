// Bytes cross the Wails bridge as base64 strings (Go []byte in JSON).

export function decodeBase64(b64: string): Uint8Array {
  const bin = atob(b64)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i)
  return out
}

// Chunked so large files do not overflow the argument limit of fromCharCode.
const CHUNK = 0x8000

export function encodeBase64(bytes: Uint8Array): string {
  let bin = ''
  for (let i = 0; i < bytes.length; i += CHUNK) {
    bin += String.fromCharCode(...bytes.subarray(i, i + CHUNK))
  }
  return btoa(bin)
}
