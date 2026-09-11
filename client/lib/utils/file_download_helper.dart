import 'file_download_helper_stub.dart'
    if (dart.library.io) 'file_download_helper_io.dart'
    if (dart.library.html) 'file_download_helper_web.dart';

/// Cross-platform file download and viewer utility.
abstract class FileDownloadHelper {
  /// Downloads and opens the given byte array on the current platform.
  static Future<void> downloadAndOpenFile({
    required List<int> bytes,
    required String fileName,
  }) =>
      downloadAndOpenFileImpl(bytes: bytes, fileName: fileName);
}
