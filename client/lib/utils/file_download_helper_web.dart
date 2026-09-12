import 'dart:js_interop';
import 'dart:typed_data';
import 'package:web/web.dart' as web;

String _inferMimeType(String fileName) {
  final lower = fileName.toLowerCase();
  if (lower.endsWith('.json')) return 'application/json';
  if (lower.endsWith('.txt')) return 'text/plain;charset=utf-8';
  if (lower.endsWith('.pdf')) return 'application/pdf';
  if (lower.endsWith('.csv')) return 'text/csv;charset=utf-8';
  return 'application/octet-stream';
}

/// Web implementation of file downloading using browser Blob and HTMLAnchorElement.
Future<void> downloadAndOpenFileImpl({
  required List<int> bytes,
  required String fileName,
  String? mimeType,
}) async {
  final uint8List = Uint8List.fromList(bytes);
  final resolvedMime = mimeType ?? _inferMimeType(fileName);
  final blob = web.Blob([uint8List.toJS].toJS, web.BlobPropertyBag(type: resolvedMime));
  final objectUrl = web.URL.createObjectURL(blob);
  final anchor = web.document.createElement('a') as web.HTMLAnchorElement;
  anchor.href = objectUrl;
  anchor.download = fileName;
  web.document.body?.appendChild(anchor);
  anchor.click();
  web.document.body?.removeChild(anchor);
  web.URL.revokeObjectURL(objectUrl);
}
