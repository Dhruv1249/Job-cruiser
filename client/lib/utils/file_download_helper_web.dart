import 'dart:js_interop';
import 'dart:typed_data';
import 'package:web/web.dart' as web;

/// Web implementation of file downloading using browser Blob and HTMLAnchorElement.
Future<void> downloadAndOpenFileImpl({
  required List<int> bytes,
  required String fileName,
}) async {
  final uint8List = Uint8List.fromList(bytes);
  final blob = web.Blob([uint8List.toJS].toJS, web.BlobPropertyBag(type: 'application/pdf'));
  final objectUrl = web.URL.createObjectURL(blob);
  final anchor = web.document.createElement('a') as web.HTMLAnchorElement;
  anchor.href = objectUrl;
  anchor.download = fileName;
  web.document.body?.appendChild(anchor);
  anchor.click();
  web.document.body?.removeChild(anchor);
  web.URL.revokeObjectURL(objectUrl);
}
