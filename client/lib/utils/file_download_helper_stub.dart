/// Fallback implementation of file downloading for unsupported platforms.
Future<void> downloadAndOpenFileImpl({
  required List<int> bytes,
  required String fileName,
}) async {
  throw UnsupportedError('Platform not supported for file download');
}
