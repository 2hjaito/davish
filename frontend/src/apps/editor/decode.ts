// Turning file bytes into editor text, refusing what is not text.

// A NUL in the first block is the usual sign of a binary file.
const SNIFF = 8192

/**
 * The file as UTF-8 text, or null for binary files and other encodings.
 * A byte order mark is kept, so saving writes the file back as it was.
 */
export function decodeText(bytes: Uint8Array): string | null {
  if (bytes.subarray(0, SNIFF).includes(0)) return null
  try {
    return new TextDecoder('utf-8', { fatal: true, ignoreBOM: true }).decode(bytes)
  } catch {
    return null
  }
}
