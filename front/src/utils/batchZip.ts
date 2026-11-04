// Dependency-free ZIP (STORE mode) for offline self-contained HTML reports.
// UTF-8 filenames; no compression means the reports remain byte-for-byte intact.
const enc = new TextEncoder();
const crcTable = new Uint32Array(256);
for (let n = 0; n < 256; n++) {
  let c = n;
  for (let k = 0; k < 8; k++) c = c & 1 ? (0xedb88320 ^ (c >>> 1)) : (c >>> 1);
  crcTable[n] = c >>> 0;
}
function crc32(bytes: Uint8Array): number {
  let crc = 0xffffffff;
  for (const b of bytes) crc = crcTable[(crc ^ b) & 255] ^ (crc >>> 8);
  return (crc ^ 0xffffffff) >>> 0;
}
function header(length: number): { bytes: Uint8Array; view: DataView } {
  const bytes = new Uint8Array(length);
  return { bytes, view: new DataView(bytes.buffer) };
}
export function makeReportZip(files: { name: string; html: string }[]): Blob {
  const locals: Uint8Array[] = [], central: Uint8Array[] = [];
  let offset = 0;
  for (const file of files) {
    const name = enc.encode(file.name), data = enc.encode(file.html), crc = crc32(data);
    if (name.length > 65535 || data.length > 0xffffffff) throw new Error('Report too large for ZIP');
    const lh = header(30 + name.length); const v = lh.view;
    v.setUint32(0, 0x04034b50, true); v.setUint16(4, 20, true);
    v.setUint16(6, 0x0800, true); v.setUint16(8, 0, true);
    v.setUint32(14, crc, true); v.setUint32(18, data.length, true); v.setUint32(22, data.length, true);
    v.setUint16(26, name.length, true); lh.bytes.set(name, 30);
    locals.push(lh.bytes, data);
    const ch = header(46 + name.length); const c = ch.view;
    c.setUint32(0, 0x02014b50, true); c.setUint16(4, 20, true); c.setUint16(6, 20, true);
    c.setUint16(8, 0x0800, true); c.setUint16(10, 0, true);
    c.setUint32(16, crc, true); c.setUint32(20, data.length, true); c.setUint32(24, data.length, true);
    c.setUint16(28, name.length, true); c.setUint32(42, offset, true);
    ch.bytes.set(name, 46); central.push(ch.bytes);
    offset += lh.bytes.length + data.length;
  }
  if (files.length > 65535 || offset > 0xffffffff) throw new Error('Too many or too-large reports');
  const directorySize = central.reduce((sum, b) => sum + b.length, 0);
  const end = header(22), e = end.view;
  e.setUint32(0, 0x06054b50, true); e.setUint16(8, files.length, true);
  e.setUint16(10, files.length, true); e.setUint32(12, directorySize, true);
  e.setUint32(16, offset, true);
  // Use BlobPart-compatible ArrayBuffers to support strict TS DOM typings.
  return new Blob([...locals, ...central, end.bytes].map(b => b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength) as ArrayBuffer), { type: 'application/zip' });
}
