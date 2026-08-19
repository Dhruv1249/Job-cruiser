import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';
import 'package:flutter/material.dart';
import 'package:open_file/open_file.dart';
import 'package:path_provider/path_provider.dart';
import 'package:syncfusion_flutter_pdfviewer/pdfviewer.dart';
import 'package:url_launcher/url_launcher.dart';
import '../main.dart' show AppColors;

/// Reusable bottom sheet displayed after a successful resume tailoring or cover letter generation.
/// Renders the compiled PDF inline using SfPdfViewer, and provides controls to open the
/// open-overleaf IDE and download the PDF to local storage.
class TailoringResultSheet extends StatefulWidget {
  const TailoringResultSheet({
    super.key,
    required this.sessionType,
    required this.pdfBase64,
    required this.pdfWebUrl,
    required this.pageCount,
    required this.folderPath,
  });

  final String sessionType;
  final String pdfBase64;
  final String pdfWebUrl;
  final int pageCount;
  final String folderPath;

  @override
  State<TailoringResultSheet> createState() => _TailoringResultSheetState();
}

class _TailoringResultSheetState extends State<TailoringResultSheet> {
  bool _isDownloading = false;

  String get _sessionLabel =>
      widget.sessionType == 'cover_letter' ? 'Cover Letter' : 'Resume';

  Uint8List? get _pdfBytes {
    if (widget.pdfBase64.isEmpty) return null;
    try {
      return base64Decode(widget.pdfBase64);
    } catch (_) {
      return null;
    }
  }

  Future<void> _handleOpenInOverleaf() async {
    final uri = Uri.tryParse(widget.pdfWebUrl);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Could not open open-overleaf IDE')),
        );
      }
    }
  }

  Future<void> _handleDownloadPDF() async {
    final pdfBytes = _pdfBytes;
    if (pdfBytes == null || pdfBytes.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('PDF data unavailable for download')),
      );
      return;
    }

    setState(() => _isDownloading = true);

    try {
      final tempDirectory = await getTemporaryDirectory();
      final safeFileName =
          '${widget.folderPath}_${widget.sessionType}.pdf'.replaceAll('/', '_');
      final targetFile = File('${tempDirectory.path}/$safeFileName');
      await targetFile.writeAsBytes(pdfBytes);
      await OpenFile.open(targetFile.path);
    } catch (downloadError) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Download failed: $downloadError')),
        );
      }
    } finally {
      if (mounted) {
        setState(() => _isDownloading = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final pdfBytes = _pdfBytes;
    return DraggableScrollableSheet(
      initialChildSize: 0.92,
      minChildSize: 0.5,
      maxChildSize: 0.97,
      expand: false,
      builder: (sheetContext, scrollController) => Container(
        decoration: const BoxDecoration(
          color: AppColors.surface,
          borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
        ),
        child: Column(
          children: [
            Container(
              width: 40,
              height: 4,
              margin: const EdgeInsets.symmetric(vertical: 12),
              decoration: BoxDecoration(
                color: AppColors.outlineVariant,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 20),
              child: Row(
                children: [
                  const Icon(Icons.check_circle, color: Colors.green, size: 20),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          '$_sessionLabel Compiled Successfully',
                          style: const TextStyle(
                            fontSize: 16,
                            fontWeight: FontWeight.w700,
                            color: AppColors.primary,
                          ),
                        ),
                        Text(
                          '${widget.pageCount} page(s) · ${widget.folderPath}',
                          style: const TextStyle(
                            fontSize: 12,
                            color: AppColors.onSurfaceVariant,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close, color: AppColors.onSurfaceVariant),
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                ],
              ),
            ),
            const Divider(height: 1, color: AppColors.outlineVariant),
            Expanded(
              child: pdfBytes != null && pdfBytes.isNotEmpty
                  ? SfPdfViewer.memory(
                      pdfBytes,
                      canShowScrollHead: false,
                      canShowScrollStatus: false,
                    )
                  : const Center(
                      child: Column(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          Icon(Icons.picture_as_pdf_outlined,
                              size: 48, color: AppColors.onSurfaceVariant),
                          SizedBox(height: 12),
                          Text(
                            'PDF preview unavailable.\nOpen in overleaf to view.',
                            textAlign: TextAlign.center,
                            style: TextStyle(color: AppColors.onSurfaceVariant),
                          ),
                        ],
                      ),
                    ),
            ),
            Container(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
              decoration: const BoxDecoration(
                color: AppColors.surface,
                border: Border(
                  top: BorderSide(color: AppColors.outlineVariant),
                ),
              ),
              child: Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      onPressed: _handleOpenInOverleaf,
                      icon: const Icon(Icons.edit_document, size: 18),
                      label: const Text('Open in Overleaf'),
                      style: OutlinedButton.styleFrom(
                        foregroundColor: AppColors.primary,
                        side: const BorderSide(color: AppColors.outlineVariant),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: ElevatedButton.icon(
                      onPressed: _isDownloading ? null : _handleDownloadPDF,
                      icon: _isDownloading
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(
                                strokeWidth: 2,
                                color: Colors.white,
                              ),
                            )
                          : const Icon(Icons.download, size: 18),
                      label: Text(_isDownloading ? 'Saving...' : 'Download PDF'),
                      style: ElevatedButton.styleFrom(
                        backgroundColor: AppColors.primary,
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

/// Displays the TailoringResultSheet as a modal bottom sheet.
Future<void> showTailoringResultSheet({
  required BuildContext context,
  required String sessionType,
  required Map<String, dynamic> tailoringResponse,
}) {
  final pdfBase64 = tailoringResponse['pdf_base64'] as String? ?? '';
  final pdfWebUrl = tailoringResponse['pdf_web_url'] as String? ?? '';
  final folderPath = tailoringResponse['folder_path'] as String? ?? '';
  final pageCount = tailoringResponse['page_count'] as int? ?? 1;

  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => TailoringResultSheet(
      sessionType: sessionType,
      pdfBase64: pdfBase64,
      pdfWebUrl: pdfWebUrl,
      pageCount: pageCount,
      folderPath: folderPath,
    ),
  );
}
