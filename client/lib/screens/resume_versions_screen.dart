import 'dart:async';
import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../services/api_service.dart';
import '../widgets/tailoring_result_sheet.dart';

/// Screen displaying all AI-tailored resume versions for the authenticated user.
/// Allows viewing compiled PDFs, setting default versions, and deleting old versions.
class ResumeVersionsScreen extends StatefulWidget {
  const ResumeVersionsScreen({super.key});

  @override
  State<ResumeVersionsScreen> createState() => _ResumeVersionsScreenState();
}

class _ResumeVersionsScreenState extends State<ResumeVersionsScreen> {
  final ApiService _apiService = ApiService();
  List<Map<String, dynamic>> _versions = [];
  bool _isLoading = true;
  String? _loadingVersionId;
  Timer? _pollingTimer;

  @override
  void initState() {
    super.initState();
    _loadResumeVersions();
  }

  @override
  void dispose() {
    _pollingTimer?.cancel();
    super.dispose();
  }

  void _checkAndStartPolling() {
    final hasGenerating = _versions.any((v) => v['status'] == 'generating' || v['status'] == 'processing');
    if (hasGenerating && _pollingTimer == null) {
      _pollingTimer = Timer.periodic(const Duration(seconds: 4), (_) async {
        final versions = await _apiService.fetchResumeVersions();
        if (!mounted) return;
        setState(() {
          _versions = versions;
        });
        final stillGenerating = versions.any((v) => v['status'] == 'generating' || v['status'] == 'processing');
        if (!stillGenerating) {
          _pollingTimer?.cancel();
          _pollingTimer = null;
        }
      });
    } else if (!hasGenerating && _pollingTimer != null) {
      _pollingTimer?.cancel();
      _pollingTimer = null;
    }
  }

  Future<void> _loadResumeVersions() async {
    setState(() => _isLoading = true);
    final versions = await _apiService.fetchResumeVersions();
    if (mounted) {
      setState(() {
        _versions = versions;
        _isLoading = false;
      });
      _checkAndStartPolling();
    }
  }

  Future<void> _handleViewVersion(Map<String, dynamic> version) async {
    final versionId = version['id'] as String? ?? '';
    final status = version['status'] as String? ?? 'ready';
    final errorMessage = version['error_message'] as String? ?? '';

    if (status == 'generating' || status == 'processing') {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('AI is tailoring this resume in the background. It will be ready in a moment.'),
          backgroundColor: Colors.orange,
        ),
      );
      return;
    }

    if (status == 'failed') {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Generation failed: ${errorMessage.isNotEmpty ? errorMessage : "Unknown error"}'),
          backgroundColor: Colors.redAccent,
        ),
      );
      return;
    }

    if (versionId.isEmpty) return;

    setState(() => _loadingVersionId = versionId);

    try {
      final pdfResponse = await _apiService.fetchResumeVersionPDF(versionId);
      if (!mounted) return;

      if (pdfResponse != null && pdfResponse['pdf_base64'] != null) {
        await showTailoringResultSheet(
          context: context,
          sessionType: 'resume',
          tailoringResponse: {
            'pdf_base64': pdfResponse['pdf_base64'],
            'pdf_web_url': version['pdf_url'] ?? '',
            'folder_path': version['overleaf_folder_path'] ?? '',
            'page_count': pdfResponse['page_count'] ?? version['page_count'] ?? 1,
          },
        );
      } else {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Failed to load PDF from open-overleaf.'),
            backgroundColor: Colors.redAccent,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _loadingVersionId = null);
    }
  }

  Future<void> _handleSetDefault(String versionId) async {
    final success = await _apiService.setDefaultResumeVersion(versionId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Default resume updated.'),
          backgroundColor: AppColors.successGreen,
        ),
      );
      _loadResumeVersions();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to set default resume.'),
          backgroundColor: Colors.redAccent,
        ),
      );
    }
  }

  Future<void> _handleDelete(String versionId) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text('Delete Resume Version'),
        content: const Text(
          'Remove this version reference from Job Cruiser? The files will remain preserved in your open-overleaf project.',
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

    final success = await _apiService.deleteResumeVersion(versionId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Resume version removed.')),
      );
      _loadResumeVersions();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to delete resume version.'),
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
          'Tailored Resumes',
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
                  onRefresh: _loadResumeVersions,
                  child: ListView.separated(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 20),
                    itemCount: _versions.length,
                    separatorBuilder: (separatorContext, index) => const SizedBox(height: 12),
                    itemBuilder: (itemContext, index) => _buildVersionCard(_versions[index]),
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
                Icons.description_outlined,
                size: 40,
                color: AppColors.outline,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'No Tailored Resumes Yet',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Browse jobs in your feed and tap "Tailor Resume" to generate an AI-tailored resume compiled in open-overleaf.',
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

  Widget _buildVersionCard(Map<String, dynamic> version) {
    final versionId = version['id'] as String? ?? '';
    final label = version['label'] as String? ?? 'Untitled Resume';
    final folderPath = version['overleaf_folder_path'] as String? ?? '';
    final pageCount = version['page_count'] as int? ?? 1;
    final isDefault = version['is_default'] as bool? ?? false;
    final status = version['status'] as String? ?? 'ready';
    final errorMessage = version['error_message'] as String? ?? '';
    final createdAt = version['created_at'] as String? ?? '';
    final isCardLoading = _loadingVersionId == versionId;
    final isGenerating = status == 'generating' || status == 'processing';
    final isFailed = status == 'failed';

    final formattedDate = createdAt.length >= 10 ? createdAt.substring(0, 10) : createdAt;

    return Container(
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isDefault
              ? AppColors.matchGreen
              : isGenerating
                  ? Colors.orange.withValues(alpha: 0.5)
                  : isFailed
                      ? Colors.redAccent.withValues(alpha: 0.5)
                      : AppColors.outlineVariant,
          width: (isDefault || isGenerating || isFailed) ? 1.5 : 1.0,
        ),
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
          onTap: isCardLoading ? null : () => _handleViewVersion(version),
          child: Padding(
            padding: const EdgeInsets.all(16),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Container(
                  padding: const EdgeInsets.all(10),
                  decoration: BoxDecoration(
                    color: isDefault
                        ? AppColors.matchGreen.withValues(alpha: 0.1)
                        : isGenerating
                            ? Colors.orange.withValues(alpha: 0.1)
                            : isFailed
                                ? Colors.redAccent.withValues(alpha: 0.1)
                                : AppColors.surfaceContainerHigh,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: isGenerating
                      ? const SizedBox(
                          width: 24,
                          height: 24,
                          child: CircularProgressIndicator(strokeWidth: 2.5, color: Colors.orange),
                        )
                      : Icon(
                          isFailed
                              ? Icons.error_outline
                              : Icons.description,
                          color: isDefault
                              ? AppColors.matchGreen
                              : isFailed
                                  ? Colors.redAccent
                                  : AppColors.primary,
                          size: 24,
                        ),
                ),
                const SizedBox(width: 14),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(
                        children: [
                          Expanded(
                            child: Text(
                              label,
                              style: const TextStyle(
                                fontSize: 15,
                                fontWeight: FontWeight.w700,
                                color: AppColors.primary,
                              ),
                              maxLines: 2,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                          if (isGenerating)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: Colors.orange.withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: const Text(
                                'Generating...',
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600,
                                  color: Colors.orange,
                                ),
                              ),
                            )
                          else if (isFailed)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: Colors.redAccent.withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: const Text(
                                'Failed',
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600,
                                  color: Colors.redAccent,
                                ),
                              ),
                            )
                          else if (isDefault)
                            Container(
                              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                              decoration: BoxDecoration(
                                color: AppColors.matchGreen.withValues(alpha: 0.15),
                                borderRadius: BorderRadius.circular(12),
                              ),
                              child: const Text(
                                'Default',
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w600,
                                  color: AppColors.matchGreen,
                                ),
                              ),
                            ),
                        ],
                      ),
                      const SizedBox(height: 6),
                      if (isGenerating)
                        const Text(
                          'AI is tailoring and compiling LaTeX in Open-Overleaf...',
                          style: TextStyle(
                            fontSize: 12,
                            color: Colors.orange,
                            fontWeight: FontWeight.w500,
                          ),
                        )
                      else if (isFailed)
                        Text(
                          errorMessage.isNotEmpty ? errorMessage : 'Tailoring failed.',
                          style: const TextStyle(
                            fontSize: 12,
                            color: Colors.redAccent,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        )
                      else ...[
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
                        _handleViewVersion(version);
                      } else if (action == 'default') {
                        _handleSetDefault(versionId);
                      } else if (action == 'delete') {
                        _handleDelete(versionId);
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
                      if (!isDefault)
                        const PopupMenuItem(
                          value: 'default',
                          child: Row(
                            children: [
                              Icon(Icons.star_outline, size: 18),
                              SizedBox(width: 8),
                              Text('Set as Default'),
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
