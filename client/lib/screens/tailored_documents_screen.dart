import 'dart:async';
import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../services/api_service.dart';
import '../widgets/tailoring_result_sheet.dart';
import '../widgets/company_logo_avatar.dart';

/// Represents a single tailored job entity grouping its tailored resume and cover letter.
class TailoredJobDocumentGroup {
  const TailoredJobDocumentGroup({
    required this.groupKey,
    required this.company,
    required this.role,
    required this.folderPath,
    required this.createdAt,
    this.jobId,
    this.resumeVersion,
    this.coverLetterVersion,
  });

  final String groupKey;
  final String company;
  final String role;
  final String folderPath;
  final String createdAt;
  final String? jobId;
  final Map<String, dynamic>? resumeVersion;
  final Map<String, dynamic>? coverLetterVersion;

  bool get isGenerating {
    final resumeStatus = resumeVersion?['status'] as String? ?? '';
    final coverStatus = coverLetterVersion?['status'] as String? ?? '';
    return resumeStatus == 'generating' ||
        resumeStatus == 'processing' ||
        coverStatus == 'generating' ||
        coverStatus == 'processing';
  }

  bool get isFailed {
    final resumeStatus = resumeVersion?['status'] as String? ?? '';
    final coverStatus = coverLetterVersion?['status'] as String? ?? '';
    return resumeStatus == 'failed' || coverStatus == 'failed';
  }

  TailoredJobDocumentGroup copyWith({
    String? groupKey,
    String? company,
    String? role,
    String? folderPath,
    String? createdAt,
    String? jobId,
    Map<String, dynamic>? resumeVersion,
    Map<String, dynamic>? coverLetterVersion,
  }) {
    return TailoredJobDocumentGroup(
      groupKey: groupKey ?? this.groupKey,
      company: company ?? this.company,
      role: role ?? this.role,
      folderPath: folderPath ?? this.folderPath,
      createdAt: createdAt ?? this.createdAt,
      jobId: jobId ?? this.jobId,
      resumeVersion: resumeVersion ?? this.resumeVersion,
      coverLetterVersion: coverLetterVersion ?? this.coverLetterVersion,
    );
  }
}

/// Groups flat lists of resumes and cover letters by their associated job or folder.
List<TailoredJobDocumentGroup> groupTailoredDocuments(
  List<Map<String, dynamic>> resumes,
  List<Map<String, dynamic>> coverLetters,
) {
  final Map<String, TailoredJobDocumentGroup> groupMap = {};

  for (final resume in resumes) {
    final jobId = resume['job_id'] as String?;
    final folderPath = resume['overleaf_folder_path'] as String? ?? '';
    final label = resume['label'] as String? ?? 'Tailored Resume';
    final createdAt = resume['created_at'] as String? ?? '';

    final key = (jobId != null && jobId.isNotEmpty)
        ? jobId
        : (folderPath.isNotEmpty ? folderPath : 'resume_${resume['id']}');

    final parsed = _parseJobDetails(label, folderPath);

    groupMap[key] = TailoredJobDocumentGroup(
      groupKey: key,
      company: parsed['company']!,
      role: parsed['role']!,
      folderPath: folderPath,
      createdAt: createdAt,
      jobId: jobId,
      resumeVersion: resume,
    );
  }

  for (final cover in coverLetters) {
    final jobId = cover['job_id'] as String?;
    final folderPath = cover['overleaf_folder_path'] as String? ?? '';
    final label = cover['label'] as String? ?? 'Generated Cover Letter';
    final createdAt = cover['created_at'] as String? ?? '';

    final key = (jobId != null && jobId.isNotEmpty)
        ? jobId
        : (folderPath.isNotEmpty ? folderPath : 'cover_${cover['id']}');

    if (groupMap.containsKey(key)) {
      final existing = groupMap[key]!;
      groupMap[key] = existing.copyWith(coverLetterVersion: cover);
    } else {
      final parsed = _parseJobDetails(label, folderPath);
      groupMap[key] = TailoredJobDocumentGroup(
        groupKey: key,
        company: parsed['company']!,
        role: parsed['role']!,
        folderPath: folderPath,
        createdAt: createdAt,
        jobId: jobId,
        coverLetterVersion: cover,
      );
    }
  }

  final sortedList = groupMap.values.toList();
  sortedList.sort((first, second) => second.createdAt.compareTo(first.createdAt));
  return sortedList;
}

Map<String, String> _parseJobDetails(String label, String folderPath) {
  if (label.contains(' — ')) {
    final parts = label.split(' — ');
    final companyName = parts[0].trim();
    final remaining = parts.length > 1 ? parts.sublist(1).join(' — ') : '';
    final cleanedRole = remaining
        .replaceAll(RegExp(r'\s*\((?:Cover Letter\s*)?\d{4}-\d{2}-\d{2}\)'), '')
        .replaceAll(RegExp(r'\s*\(Cover Letter\)', caseSensitive: false), '')
        .trim();
    return {
      'company': companyName.isNotEmpty ? companyName : 'Target Company',
      'role': cleanedRole.isNotEmpty ? cleanedRole : 'Tailored Position',
    };
  }

  if (folderPath.isNotEmpty) {
    final folderClean = folderPath.replaceAll('job_applications/', '');
    final segments = folderClean.split('_');
    if (segments.length >= 2) {
      return {
        'company': segments[0],
        'role': segments[1],
      };
    }
  }

  return {
    'company': 'General Master Profile',
    'role': label,
  };
}

/// Interactive expandable card representing a single job with its tailored CV and Cover Letter.
class TailoredJobCard extends StatefulWidget {
  const TailoredJobCard({
    super.key,
    required this.group,
    required this.onViewDocument,
    required this.onDeleteDocument,
    required this.onSetDefaultResume,
  });

  final TailoredJobDocumentGroup group;
  final Function(Map<String, dynamic> doc, String type) onViewDocument;
  final Function(String docId, String type) onDeleteDocument;
  final Function(String docId) onSetDefaultResume;

  @override
  State<TailoredJobCard> createState() => _TailoredJobCardState();
}

class _TailoredJobCardState extends State<TailoredJobCard> {
  bool _isExpanded = false;

  @override
  Widget build(BuildContext context) {
    final group = widget.group;
    final formattedDate = group.createdAt.length >= 10
        ? group.createdAt.substring(0, 10)
        : group.createdAt;

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: group.isGenerating
              ? Colors.orange.withValues(alpha: 0.5)
              : group.isFailed
                  ? Colors.redAccent.withValues(alpha: 0.5)
                  : AppColors.outlineVariant,
          width: (group.isGenerating || group.isFailed) ? 1.5 : 1.0,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.03),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Column(
        children: [
          InkWell(
            borderRadius: BorderRadius.circular(12),
            onTap: () {
              setState(() {
                _isExpanded = !_isExpanded;
              });
            },
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: Row(
                children: [
                  CompanyLogoAvatar(
                    companyName: group.company,
                    jobUrl: '',
                    size: 38,
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          group.role,
                          style: const TextStyle(
                            fontSize: 15,
                            fontWeight: FontWeight.w700,
                            color: AppColors.primary,
                          ),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 3),
                        Text(
                          '${group.company} • $formattedDate',
                          style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w500,
                            color: AppColors.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  if (group.isGenerating)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                      margin: const EdgeInsets.only(right: 8),
                      decoration: BoxDecoration(
                        color: Colors.orange.withValues(alpha: 0.15),
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: const Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          SizedBox(
                            width: 12,
                            height: 12,
                            child: CircularProgressIndicator(
                              strokeWidth: 2,
                              color: Colors.orange,
                            ),
                          ),
                          SizedBox(width: 6),
                          Text(
                            'Tailoring...',
                            style: TextStyle(
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              color: Colors.orange,
                            ),
                          ),
                        ],
                      ),
                    )
                  else if (group.isFailed)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                      margin: const EdgeInsets.only(right: 8),
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
                    ),
                  Icon(
                    _isExpanded ? Icons.keyboard_arrow_up : Icons.keyboard_arrow_down,
                    color: AppColors.outline,
                    size: 22,
                  ),
                ],
              ),
            ),
          ),
          if (_isExpanded) ...[
            const Divider(height: 1, color: AppColors.outlineVariant),
            Container(
              padding: const EdgeInsets.all(12),
              color: AppColors.surfaceContainerLow.withValues(alpha: 0.5),
              child: Column(
                children: [
                  if (group.resumeVersion != null)
                    _buildDocumentTile(
                      icon: Icons.description,
                      iconColor: AppColors.primary,
                      title: 'Tailored CV / Resume',
                      doc: group.resumeVersion!,
                      type: 'resume',
                      isDefault: group.resumeVersion!['is_default'] as bool? ?? false,
                    )
                  else
                    _buildMissingDocumentTile('CV / Resume not generated for this application'),
                  const SizedBox(height: 8),
                  if (group.coverLetterVersion != null)
                    _buildDocumentTile(
                      icon: Icons.mail,
                      iconColor: const Color(0xFF6366F1),
                      title: 'Generated Cover Letter',
                      doc: group.coverLetterVersion!,
                      type: 'cover_letter',
                      isDefault: false,
                    )
                  else
                    _buildMissingDocumentTile('Cover Letter not generated for this application'),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildMissingDocumentTile(String message) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
      ),
      child: Row(
        children: [
          const Icon(Icons.info_outline, size: 18, color: AppColors.outline),
          const SizedBox(width: 10),
          Expanded(
            child: Text(
              message,
              style: const TextStyle(
                fontSize: 12,
                fontStyle: FontStyle.italic,
                color: AppColors.onSurfaceVariant,
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildDocumentTile({
    required IconData icon,
    required Color iconColor,
    required String title,
    required Map<String, dynamic> doc,
    required String type,
    required bool isDefault,
  }) {
    final status = doc['status'] as String? ?? 'ready';
    final pageCount = doc['page_count'] as int? ?? 1;
    final docId = doc['id'] as String? ?? '';
    final isGenerating = status == 'generating' || status == 'processing';
    final isFailed = status == 'failed';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Row(
        children: [
          Container(
            padding: const EdgeInsets.all(8),
            decoration: BoxDecoration(
              color: iconColor.withValues(alpha: 0.1),
              borderRadius: BorderRadius.circular(6),
            ),
            child: Icon(icon, color: iconColor, size: 18),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 13,
                        fontWeight: FontWeight.w700,
                        color: AppColors.primary,
                      ),
                    ),
                    if (isDefault) ...[
                      const SizedBox(width: 6),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                        decoration: BoxDecoration(
                          color: AppColors.matchGreen.withValues(alpha: 0.15),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: const Text(
                          'DEFAULT',
                          style: TextStyle(
                            fontSize: 9,
                            fontWeight: FontWeight.w800,
                            color: AppColors.matchGreen,
                          ),
                        ),
                      ),
                    ],
                  ],
                ),
                const SizedBox(height: 2),
                Text(
                  isGenerating
                      ? 'Compiling in Open-Overleaf...'
                      : isFailed
                          ? 'Compilation failed'
                          : '$pageCount page(s) · Ready',
                  style: TextStyle(
                    fontSize: 11,
                    color: isGenerating
                        ? Colors.orange
                        : isFailed
                            ? Colors.redAccent
                            : AppColors.onSurfaceVariant,
                    fontWeight: isGenerating ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(width: 8),
          FilledButton.tonal(
            onPressed: isGenerating ? null : () => widget.onViewDocument(doc, type),
            style: FilledButton.styleFrom(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              minimumSize: const Size(0, 32),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
            ),
            child: const Text('View PDF', style: TextStyle(fontSize: 12, fontWeight: FontWeight.w600)),
          ),
          PopupMenuButton<String>(
            icon: const Icon(Icons.more_vert, size: 18, color: AppColors.outline),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(),
            onSelected: (action) {
              if (action == 'view') {
                widget.onViewDocument(doc, type);
              } else if (action == 'default') {
                widget.onSetDefaultResume(docId);
              } else if (action == 'delete') {
                widget.onDeleteDocument(docId, type);
              }
            },
            itemBuilder: (popupContext) => [
              const PopupMenuItem(
                value: 'view',
                child: Row(
                  children: [
                    Icon(Icons.visibility_outlined, size: 16),
                    SizedBox(width: 8),
                    Text('View PDF', style: TextStyle(fontSize: 13)),
                  ],
                ),
              ),
              if (type == 'resume' && !isDefault)
                const PopupMenuItem(
                  value: 'default',
                  child: Row(
                    children: [
                      Icon(Icons.star_outline, size: 16),
                      SizedBox(width: 8),
                      Text('Set as Default', style: TextStyle(fontSize: 13)),
                    ],
                  ),
                ),
              const PopupMenuItem(
                value: 'delete',
                child: Row(
                  children: [
                    Icon(Icons.delete_outline, color: AppColors.error, size: 16),
                    SizedBox(width: 8),
                    Text('Delete', style: TextStyle(color: AppColors.error, fontSize: 13)),
                  ],
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }
}

/// Standalone screen rendering all tailored applications grouped by job with dropdown documents.
class TailoredDocumentsScreen extends StatefulWidget {
  const TailoredDocumentsScreen({super.key});

  @override
  State<TailoredDocumentsScreen> createState() => _TailoredDocumentsScreenState();
}

class _TailoredDocumentsScreenState extends State<TailoredDocumentsScreen> {
  final ApiService _apiService = ApiService();
  List<TailoredJobDocumentGroup> _groups = [];
  bool _isLoading = true;
  Timer? _pollingTimer;

  @override
  void initState() {
    super.initState();
    _loadDocuments();
  }

  @override
  void dispose() {
    _pollingTimer?.cancel();
    super.dispose();
  }

  void _checkAndStartPolling() {
    final hasGenerating = _groups.any((g) => g.isGenerating);
    if (hasGenerating && _pollingTimer == null) {
      _pollingTimer = Timer.periodic(const Duration(seconds: 4), (_) async {
        final results = await Future.wait([
          _apiService.fetchResumeVersions(),
          _apiService.fetchCoverLetterVersions(),
        ]);
        if (!mounted) return;
        final updatedGroups = groupTailoredDocuments(results[0], results[1]);
        setState(() {
          _groups = updatedGroups;
        });
        final stillGenerating = updatedGroups.any((g) => g.isGenerating);
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

  Future<void> _loadDocuments() async {
    setState(() => _isLoading = true);
    final results = await Future.wait([
      _apiService.fetchResumeVersions(),
      _apiService.fetchCoverLetterVersions(),
    ]);
    if (!mounted) return;
    setState(() {
      _groups = groupTailoredDocuments(results[0], results[1]);
      _isLoading = false;
    });
    _checkAndStartPolling();
  }

  Future<void> _handleViewDocument(Map<String, dynamic> doc, String type) async {
    final docId = doc['id'] as String? ?? '';
    final status = doc['status'] as String? ?? 'ready';
    final errorMessage = doc['error_message'] as String? ?? '';

    if (status == 'generating' || status == 'processing') {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('AI is tailoring this document in the background. It will be ready shortly.'),
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

    if (docId.isEmpty) return;

    final pdfResponse = type == 'resume'
        ? await _apiService.fetchResumeVersionPDF(docId)
        : await _apiService.fetchCoverLetterPDF(docId);

    if (!mounted) return;

    if (pdfResponse != null && pdfResponse['pdf_base64'] != null) {
      await showTailoringResultSheet(
        context: context,
        sessionType: type,
        tailoringResponse: {
          'pdf_base64': pdfResponse['pdf_base64'],
          'pdf_web_url': doc['pdf_url'] ?? '',
          'folder_path': doc['overleaf_folder_path'] ?? '',
          'page_count': pdfResponse['page_count'] ?? doc['page_count'] ?? 1,
        },
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to load PDF from Open-Overleaf.'),
          backgroundColor: Colors.redAccent,
        ),
      );
    }
  }

  Future<void> _handleSetDefault(String resumeId) async {
    final success = await _apiService.setDefaultResumeVersion(resumeId);
    if (!mounted) return;
    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Default resume updated.'),
          backgroundColor: AppColors.successGreen,
        ),
      );
      _loadDocuments();
    }
  }

  Future<void> _handleDeleteDocument(String docId, String type) async {
    final isResume = type == 'resume';
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(isResume ? 'Delete Resume Version' : 'Delete Cover Letter'),
        content: const Text(
          'Remove this version reference from Job Cruiser? The files will remain preserved in your Open-Overleaf project.',
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

    final success = isResume
        ? await _apiService.deleteResumeVersion(docId)
        : await _apiService.deleteCoverLetterVersion(docId);

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${isResume ? "Resume" : "Cover letter"} version removed.')),
      );
      _loadDocuments();
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
          'Tailored Application Documents',
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
          : _groups.isEmpty
              ? _buildEmptyState()
              : RefreshIndicator(
                  onRefresh: _loadDocuments,
                  child: ListView.builder(
                    padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 20),
                    itemCount: _groups.length,
                    itemBuilder: (itemContext, index) => TailoredJobCard(
                      group: _groups[index],
                      onViewDocument: _handleViewDocument,
                      onDeleteDocument: _handleDeleteDocument,
                      onSetDefaultResume: _handleSetDefault,
                    ),
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
              decoration: const BoxDecoration(
                color: AppColors.surfaceContainerHigh,
                shape: BoxShape.circle,
              ),
              child: const Icon(
                Icons.folder_shared_outlined,
                size: 40,
                color: AppColors.outline,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'No Tailored Documents Yet',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Browse jobs in your feed and tap "Tailor Application" to generate an ATS-optimized CV and Cover Letter together.',
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
}
