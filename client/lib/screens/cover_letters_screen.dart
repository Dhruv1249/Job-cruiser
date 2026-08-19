import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../services/api_service.dart';
import '../widgets/tailoring_result_sheet.dart';

/// Screen displaying all AI-generated cover letters for the authenticated user.
/// Allows viewing compiled PDFs and deleting cover letter records.
class CoverLettersScreen extends StatefulWidget {
  const CoverLettersScreen({super.key});

  @override
  State<CoverLettersScreen> createState() => _CoverLettersScreenState();
}

class _CoverLettersScreenState extends State<CoverLettersScreen> {
  final ApiService _apiService = ApiService();
  List<Map<String, dynamic>> _versions = [];
  bool _isLoading = true;
  String? _loadingVersionId;

  @override
  void initState() {
    super.initState();
    _loadCoverLetters();
  }

  Future<void> _loadCoverLetters() async {
    setState(() => _isLoading = true);
    final versions = await _apiService.fetchCoverLetterVersions();
    if (mounted) {
      setState(() {
        _versions = versions;
        _isLoading = false;
      });
    }
  }

  Future<void> _handleViewCoverLetter(Map<String, dynamic> item) async {
    final coverLetterId = item['id'] as String? ?? '';
    if (coverLetterId.isEmpty) return;

    setState(() => _loadingVersionId = coverLetterId);

    try {
      final pdfResponse = await _apiService.fetchCoverLetterPDF(coverLetterId);
      if (!mounted) return;

      if (pdfResponse != null && pdfResponse['pdf_base64'] != null) {
        await showTailoringResultSheet(
          context: context,
          sessionType: 'cover_letter',
          tailoringResponse: {
            'pdf_base64': pdfResponse['pdf_base64'],
            'pdf_web_url': item['pdf_url'] ?? '',
            'folder_path': item['overleaf_folder_path'] ?? '',
            'page_count': pdfResponse['page_count'] ?? item['page_count'] ?? 1,
          },
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Failed to load cover letter PDF from open-overleaf.'),
            backgroundColor: Colors.redAccent,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _loadingVersionId = null);
    }
  }

  Future<void> _handleDelete(String coverLetterId) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Delete Cover Letter'),
        content: const Text(
          'Remove this cover letter reference from Job Cruiser? The files will remain preserved in your open-overleaf project.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            style: TextButton.styleFrom(foregroundColor: AppColors.error),
            child: const Text('Delete'),
          ),
        ],
      ),
    );

    if (confirmed != true) return;

    final success = await _apiService.deleteCoverLetterVersion(coverLetterId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Cover letter removed.')),
      );
      _loadCoverLetters();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to delete cover letter.'),
          backgroundColor: Colors.redAccent,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: AppColors.surface,
      appBar: AppBar(
        backgroundColor: AppColors.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: const Text(
          'Generated Cover Letters',
          style: TextStyle(
            color: AppColors.primary,
            fontWeight: FontWeight.w700,
            fontFamily: 'Inter',
          ),
        ),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(1),
          child: Container(color: AppColors.outlineVariant, height: 1),
        ),
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator(color: AppColors.primary))
          : _versions.isEmpty
              ? _buildEmptyState()
              : RefreshIndicator(
                  onRefresh: _loadCoverLetters,
                  child: ListView.separated(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 20),
                    itemCount: _versions.length,
                    separatorBuilder: (separatorContext, index) => const SizedBox(height: 12),
                    itemBuilder: (itemContext, index) => _buildCard(_versions[index]),
                  ),
                ),
    );
  }

  Widget _buildEmptyState() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.all(20),
              decoration: BoxDecoration(
                color: AppColors.surfaceContainerHigh,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.mail_outline,
                size: 40,
                color: AppColors.outline,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'No Cover Letters Yet',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Browse jobs in your feed and tap "Cover Letter" to generate an AI cover letter tailored to the job description.',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 14,
                color: AppColors.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCard(Map<String, dynamic> item) {
    final coverLetterId = item['id'] as String? ?? '';
    final label = item['label'] as String? ?? 'Untitled Cover Letter';
    final folderPath = item['overleaf_folder_path'] as String? ?? '';
    final pageCount = item['page_count'] as int? ?? 1;
    final createdAt = item['created_at'] as String? ?? '';
    final isCardLoading = _loadingVersionId == coverLetterId;

    final formattedDate = createdAt.length >= 10 ? createdAt.substring(0, 10) : createdAt;

    return Container(
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant, width: 1.0),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.03),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: isCardLoading ? null : () => _handleViewCoverLetter(item),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: AppColors.surfaceContainerHigh,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: const Icon(
                    Icons.mail,
                    color: AppColors.primary,
                    size: 24,
                  ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        label,
                        style: const TextStyle(
                          fontSize: 15,
                          fontWeight: FontWeight.w700,
                          color: AppColors.primary,
                        ),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 6),
                      Text(
                        'Folder: $folderPath',
                        style: const TextStyle(
                          fontSize: 12,
                          color: AppColors.onSurfaceVariant,
                          fontFamily: 'monospace',
                        ),
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                      ),
                      const SizedBox(height: 4),
                      Text(
                        '$pageCount page(s) · $formattedDate',
                        style: const TextStyle(
                          fontSize: 12,
                          color: AppColors.outline,
                        ),
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                if (isCardLoading)
                  const SizedBox(
                    width: 24,
                    height: 24,
                    child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.primary),
                  )
                else
                  PopupMenuButton<String>(
                    icon: const Icon(Icons.more_vert, color: AppColors.outline, size: 20),
                    onSelected: (action) {
                      if (action == 'view') {
                        _handleViewCoverLetter(item);
                      } else if (action == 'delete') {
                        _handleDelete(coverLetterId);
                      }
                    },
                    itemBuilder: (popupContext) => [
                      const PopupMenuItem(
                        value: 'view',
                        child: Row(
                          children: [
                            Icon(Icons.visibility_outlined, size: 18),
                            SizedBox(width: 8),
                            Text('View PDF'),
                          ],
                        ),
                      ),
                      const PopupMenuItem(
                        value: 'delete',
                        child: Row(
                          children: [
                            Icon(Icons.delete_outline, color: AppColors.error, size: 18),
                            SizedBox(width: 8),
                            Text('Delete', style: TextStyle(color: AppColors.error)),
                          ],
                        ),
                      ),
                    ],
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
