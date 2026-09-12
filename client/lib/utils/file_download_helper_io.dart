import 'dart:io';
import 'package:open_file/open_file.dart';
import 'package:path_provider/path_provider.dart';

/// Native mobile and desktop implementation of file downloading using temporary files.
Future<void> downloadAndOpenFileImpl({
  required List<int> bytes,
  required String fileName,
  String? mimeType,
}) async {
  final temporaryDirectory = await getTemporaryDirectory();
  final targetFile = File('${temporaryDirectory.path}/$fileName');
  await targetFile.writeAsBytes(bytes);
  await OpenFile.open(targetFile.path, type: mimeType);
}
