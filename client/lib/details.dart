import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import 'main.dart' show AppColors;
import 'models/job.dart';
import 'services/api_service.dart';
import 'widgets/job_description_renderer.dart';
import 'widgets/company_logo_avatar.dart';
import 'widgets/tailoring_result_sheet.dart';
import 'preferences.dart' as preferences_page;

void main() {
  runApp(const CompanyDetailsApp());
}

class CompanyDetailsApp extends StatelessWidget {
  const CompanyDetailsApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Company Deep Dive',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.background,
        fontFamily: 'Inter',
        colorScheme: ColorScheme.fromSeed(
          seedColor: AppColors.primary,
          surface: AppColors.surface,
          primary: AppColors.primary,
        ),
      ),
      home: const CompanyDetailsPage(),
    );
  }
}

class CompanyDetailsPage extends StatefulWidget {
  const CompanyDetailsPage({super.key, this.onBackToInbox, this.job});

  final VoidCallback? onBackToInbox;
  final MatchedJob? job;

  @override
  State<CompanyDetailsPage> createState() => _CompanyDetailsPageState();
}

class _CompanyDetailsPageState extends State<CompanyDetailsPage> {
  final ApiService _apiService = ApiService();
  bool _isSaving = false;
  bool _isTailoring = false;
  bool _isGeneratingCoverLetter = false;
  late String _currentStatus;

  MatchedJob? get _activeJob => widget.job;

  @override
  void initState() {
    super.initState();
    _currentStatus = widget.job?.applicationStatus ?? 'unapplied';
    final job = widget.job;
    if (job != null && job.jobId.isNotEmpty) {
      _apiService.markJobAsViewed(job.jobId);
    }
  }

  Future<void> _handleSaveStatus(String status) async {
    final activeJob = _activeJob;
    if (activeJob == null) return;

    setState(() {
      _isSaving = true;
    });

    final success = await _apiService.createApplication(
      activeJob.jobId,
      status,
    );

    if (!mounted) return;

    setState(() {
      _isSaving = false;
      if (success) {
        _currentStatus = status;
      }
    });

    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          success
              ? 'Application status updated to $status'
              : 'Failed to update application status.',
        ),
      ),
    );
  }

  Future<void> _handleDismissJob() async {
    final activeJob = _activeJob;
    if (activeJob == null) return;

    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Hide this job?'),
        content: Text(
          'Hide "${activeJob.title}" at "${activeJob.company}" from your feed? It will stay safely stored in the database, but won\'t appear in your matches.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.error,
              foregroundColor: Colors.white,
            ),
            child: const Text('Hide Job'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    final success = await _apiService.dismissJob(activeJob.jobId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Job "${activeJob.title}" hidden from feed'),
          action: SnackBarAction(
            label: 'Undo',
            textColor: Colors.amber,
            onPressed: () {
              _apiService.undismissJob(activeJob.jobId);
            },
          ),
        ),
      );
      if (widget.onBackToInbox != null) {
        widget.onBackToInbox!();
      } else {
        Navigator.maybePop(context);
      }
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to hide job. Please try again.')),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final job = _activeJob;

    if (job == null) {
      return Scaffold(
        appBar: _buildAppBar(context),
        body: const Center(
          child: Text(
            'No Job Selected',
            style: TextStyle(
              fontSize: 16,
              fontWeight: FontWeight.w600,
              color: AppColors.onSurfaceVariant,
            ),
          ),
        ),
      );
    }

    return Scaffold(
      appBar: _buildAppBar(context),
      body: SingleChildScrollView(
        padding: const EdgeInsets.only(
          left: 20,
          right: 20,
          top: 16,
          bottom: 24,
        ),
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 600),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildHeroSection(job),
                const SizedBox(height: 24),
                _buildSectionHeader('Job Description & Requirements'),
                const SizedBox(height: 16),
                _buildJobDescription(job),
                const SizedBox(height: 24),
                _buildSectionHeader('Match Analysis'),
                const SizedBox(height: 16),
                _buildMatchAnalysis(job),
              ],
            ),
          ),
        ),
      ),
      bottomNavigationBar: SafeArea(
        child: _buildBottomActionBar(),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar(BuildContext context) {
    return AppBar(
      backgroundColor: AppColors.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1),
        child: Container(color: AppColors.outlineVariant, height: 1),
      ),
      leading: IconButton(
        icon: const Icon(Icons.arrow_back, color: AppColors.primary),
        onPressed: () {
          if (widget.onBackToInbox != null) {
            widget.onBackToInbox!();
            return;
          }
          Navigator.maybePop(context);
        },
      ),
      title: Text(
        _activeJob?.company ?? 'Job Details',
        style: const TextStyle(
          color: AppColors.primary,
          fontSize: 20,
          fontWeight: FontWeight.w700,
        ),
      ),
      actions: [
        if (_activeJob != null)
          IconButton(
            icon: const Icon(Icons.visibility_off_outlined, color: AppColors.onSurfaceVariant),
            tooltip: 'Hide / Dismiss this Job',
            onPressed: _handleDismissJob,
          ),
      ],
    );
  }

  Widget _buildSectionHeader(String title) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8.0),
      child: Text(
        title,
        style: const TextStyle(
          fontSize: 18,
          fontWeight: FontWeight.w700,
          color: AppColors.primary,
        ),
      ),
    );
  }

  Future<void> _openJobUrl(String urlString) async {
    final cleanUrl = urlString.trim();
    if (cleanUrl.isEmpty) return;
    final Uri? uri = Uri.tryParse(cleanUrl);
    if (uri == null) return;

    try {
      final launched = await launchUrl(uri, mode: LaunchMode.externalApplication);
      if (!launched && mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not open URL: $cleanUrl')),
        );
      }
    } catch (_) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text('Could not open URL: $cleanUrl')),
        );
      }
    }
  }

  Widget _buildHeroSection(MatchedJob job) {
    return Container(
      decoration: BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 12,
            offset: const Offset(0, 4),
          )
        ],
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        job.title,
                        style: const TextStyle(
                          fontSize: 22,
                          fontWeight: FontWeight.w700,
                          color: AppColors.primary,
                          letterSpacing: -0.24,
                        ),
                      ),
                      const SizedBox(height: 8),
                      Row(
                        children: [
                          CompanyLogoAvatar(
                            companyName: job.company,
                            jobUrl: job.url,
                            size: 24,
                          ),
                          const SizedBox(width: 8),
                          Expanded(
                            child: Text(
                              job.company,
                              style: const TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                                color: AppColors.onSurfaceVariant,
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                        ],
                      ),
                    ],
                  ),
                ),
                Column(
                  crossAxisAlignment: CrossAxisAlignment.end,
                  children: [
                    Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 12,
                        vertical: 6,
                      ),
                      decoration: BoxDecoration(
                        color: job.matchScore >= 80
                            ? AppColors.matchGreen
                            : job.matchScore >= 60
                                ? AppColors.primaryContainer
                                : AppColors.surfaceContainerHigh,
                        borderRadius: BorderRadius.circular(16),
                      ),
                      child: Row(
                        mainAxisSize: MainAxisSize.min,
                        children: [
                          const Icon(Icons.bolt, color: Colors.white, size: 14),
                          const SizedBox(width: 4),
                          Text(
                            '${job.matchScore}% Match',
                            style: const TextStyle(
                              color: Colors.white,
                              fontSize: 12,
                              fontWeight: FontWeight.w600,
                            ),
                          ),
                        ],
                      ),
                    ),
                    const SizedBox(height: 6),
                    Text(
                      'Status: ${_currentStatus == 'unapplied' ? 'NOT APPLIED' : _currentStatus.replaceAll('_', ' ').toUpperCase()}',
                      style: const TextStyle(
                        fontSize: 10,
                        fontWeight: FontWeight.w700,
                        color: AppColors.secondary,
                      ),
                    ),
                  ],
                ),
              ],
            ),
            const SizedBox(height: 16),
            const Divider(color: AppColors.outlineVariant),
            const SizedBox(height: 12),
            Wrap(
              spacing: 16,
              runSpacing: 8,
              children: [
                _buildInfoBadge(Icons.location_on_outlined, job.location),
                if (job.isRemote)
                  _buildInfoBadge(Icons.wifi, 'Remote Position'),
                if (job.seniority.isNotEmpty)
                  _buildInfoBadge(Icons.workspace_premium_outlined, job.seniority),
                if (job.salaryText.isNotEmpty)
                  _buildInfoBadge(Icons.attach_money, job.salaryText),
                if (job.scrapedAgoText.isNotEmpty)
                  _buildInfoBadge(Icons.schedule_outlined, 'Scraped ${job.scrapedAgoText}'),
                if (job.source.isNotEmpty)
                  _buildInfoBadge(Icons.source_outlined, 'Source: ${job.source}'),
              ],
            ),
            if (job.url.isNotEmpty) ...[
              const SizedBox(height: 16),
              OutlinedButton.icon(
                onPressed: () => _openJobUrl(job.url),
                icon: const Icon(Icons.open_in_new, size: 16),
                label: Text(
                  'View Original Listing (${job.source.isNotEmpty ? job.source : "Web"})',
                  overflow: TextOverflow.ellipsis,
                ),
                style: OutlinedButton.styleFrom(
                  foregroundColor: AppColors.primary,
                  side: const BorderSide(color: AppColors.outlineVariant),
                  padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildInfoBadge(IconData icon, String text) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, size: 16, color: AppColors.onSurfaceVariant),
        const SizedBox(width: 4),
        Flexible(
          child: Text(
            text,
            maxLines: 2,
            overflow: TextOverflow.ellipsis,
            style: const TextStyle(
              fontSize: 13,
              fontWeight: FontWeight.w500,
              color: AppColors.onSurfaceVariant,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildMatchAnalysis(MatchedJob job) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'AI Evaluation Rationale',
            style: TextStyle(
              fontSize: 14,
              fontWeight: FontWeight.w700,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(height: 8),
          Text(
            job.matchReasoning.isNotEmpty
                ? job.matchReasoning
                : 'High structural overlap with your target role and stack requirements.',
            style: const TextStyle(
              fontSize: 14,
              height: 1.5,
              color: AppColors.onSurface,
            ),
          ),
          if (job.techStack.isNotEmpty) ...[
            const SizedBox(height: 16),
            const Text(
              'Required Tech Stack',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: job.techStack
                  .map(
                    (tech) => Chip(
                      label: Text(
                        tech,
                        style: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                          color: AppColors.primary,
                        ),
                      ),
                      backgroundColor: AppColors.surfaceContainer,
                      side: const BorderSide(color: AppColors.outlineVariant),
                    ),
                  )
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildJobDescription(MatchedJob job) {
    final hasRawDesc = job.rawDescription.trim().isNotEmpty;
    final hasSummary = job.summary.trim().isNotEmpty;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (hasSummary) ...[
            const Text(
              'Summary',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 6),
            Text(
              job.summary,
              style: const TextStyle(
                fontSize: 14,
                height: 1.5,
                color: AppColors.onSurface,
              ),
            ),
            if (hasRawDesc) const Divider(height: 24, color: AppColors.outlineVariant),
          ],
          if (hasRawDesc) ...[
            if (hasSummary) const SizedBox(height: 6),
            JobDescriptionRenderer(
              content: job.rawDescription,
              onLinkTap: _openJobUrl,
            ),
          ],
          if (!hasRawDesc && !hasSummary) ...[
            const Text(
              'Full job description was not extracted cleanly.',
              style: TextStyle(
                fontSize: 14,
                height: 1.6,
                color: AppColors.onSurfaceVariant,
                fontStyle: FontStyle.italic,
              ),
            ),
            if (job.url.isNotEmpty) ...[
              const SizedBox(height: 12),
              InkWell(
                onTap: () => _openJobUrl(job.url),
                child: Text(
                  job.url,
                  style: const TextStyle(
                    fontSize: 13,
                    color: AppColors.secondary,
                    decoration: TextDecoration.underline,
                  ),
                ),
              ),
            ],
          ],
        ],
      ),
    );
  }

  void _showOverleafUnconfiguredDialog() {
    showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: const Row(
          children: [
            Icon(Icons.tune, color: AppColors.primary, size: 22),
            SizedBox(width: 10),
            Text(
              'Overleaf Setup Required',
              style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700),
            ),
          ],
        ),
        content: const Text(
          'To generate and compile LaTeX resumes and cover letters with AI, please configure your Open-Overleaf Server URL in Preferences.',
          style: TextStyle(fontSize: 14, color: AppColors.onSurfaceVariant, height: 1.4),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () {
              Navigator.of(dialogContext).pop();
              Navigator.of(context).push(
                MaterialPageRoute(
                  builder: (_) => const preferences_page.SetPreferencesScreen(),
                ),
              );
            },
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            child: const Text('Configure in Preferences'),
          ),
        ],
      ),
    );
  }

  Future<void> _handleTailorResume() async {
    final job = _activeJob;
    if (job == null) return;
    setState(() => _isTailoring = true);
    try {
      final result = await _apiService.tailorResume(jobId: job.jobId, targetPages: 1);
      if (!mounted) return;
      if (result != null && result['pdf_base64'] != null) {
        await showTailoringResultSheet(
          context: context,
          sessionType: 'resume',
          tailoringResponse: result,
        );
      } else {
        final errorMessage = result?['error'] as String? ?? '';
        final statusCode = result?['status_code'] as int? ?? 0;
        final isUnconfigured = result?['unconfigured'] == true || statusCode == 422;
        if (isUnconfigured || errorMessage.toLowerCase().contains('overleaf') || errorMessage.toLowerCase().contains('preferences') || errorMessage.toLowerCase().contains('unconfigured')) {
          _showOverleafUnconfiguredDialog();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(errorMessage.isNotEmpty ? errorMessage : 'Tailoring failed. Please try again.'),
              backgroundColor: Colors.redAccent,
              action: SnackBarAction(
                label: 'Setup',
                textColor: Colors.white,
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => const preferences_page.SetPreferencesScreen(),
                    ),
                  );
                },
              ),
            ),
          );
        }
      }
    } finally {
      if (mounted) setState(() => _isTailoring = false);
    }
  }

  Future<void> _handleGenerateCoverLetter() async {
    final job = _activeJob;
    if (job == null) return;
    setState(() => _isGeneratingCoverLetter = true);
    try {
      final result = await _apiService.generateCoverLetter(jobId: job.jobId);
      if (!mounted) return;
      if (result != null && result['pdf_base64'] != null) {
        await showTailoringResultSheet(
          context: context,
          sessionType: 'cover_letter',
          tailoringResponse: result,
        );
      } else {
        final errorMessage = result?['error'] as String? ?? '';
        final statusCode = result?['status_code'] as int? ?? 0;
        final isUnconfigured = result?['unconfigured'] == true || statusCode == 422;
        if (isUnconfigured || errorMessage.toLowerCase().contains('overleaf') || errorMessage.toLowerCase().contains('preferences') || errorMessage.toLowerCase().contains('unconfigured')) {
          _showOverleafUnconfiguredDialog();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(errorMessage.isNotEmpty ? errorMessage : 'Cover letter generation failed. Please try again.'),
              backgroundColor: Colors.redAccent,
              action: SnackBarAction(
                label: 'Setup',
                textColor: Colors.white,
                onPressed: () {
                  Navigator.of(context).push(
                    MaterialPageRoute(
                      builder: (_) => const preferences_page.SetPreferencesScreen(),
                    ),
                  );
                },
              ),
            ),
          );
        }
      }
    } finally {
      if (mounted) setState(() => _isGeneratingCoverLetter = false);
    }
  }

  Widget _buildBottomActionBar() {
    final job = _activeJob;
    final isBookmarked = _currentStatus == 'bookmarked';
    final isApplied = _currentStatus == 'applied';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        border: const Border(
          top: BorderSide(color: AppColors.outlineVariant, width: 1.0),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.05),
            blurRadius: 10,
            offset: const Offset(0, -3),
          )
        ],
      ),
      child: SafeArea(
        top: false,
        child: Row(
          children: [
            IconButton.outlined(
              onPressed: _isSaving ? null : () => _handleSaveStatus(isBookmarked ? 'unbookmarked' : 'bookmarked'),
              icon: Icon(
                isBookmarked ? Icons.bookmark : Icons.bookmark_border,
                color: isBookmarked ? AppColors.matchGreen : AppColors.primary,
                size: 20,
              ),
              style: IconButton.styleFrom(
                side: BorderSide(
                  color: isBookmarked ? AppColors.matchGreen : AppColors.outlineVariant,
                ),
                padding: const EdgeInsets.all(9),
              ),
              tooltip: isBookmarked ? 'Saved' : 'Save Job',
            ),
            const SizedBox(width: 6),
            IconButton.outlined(
              onPressed: _isSaving ? null : () => _handleSaveStatus(isApplied ? 'not_applied' : 'applied'),
              icon: Icon(
                isApplied ? Icons.check_circle : Icons.check_circle_outline,
                color: isApplied ? AppColors.matchGreen : AppColors.primary,
                size: 20,
              ),
              style: IconButton.styleFrom(
                side: BorderSide(
                  color: isApplied ? AppColors.matchGreen : AppColors.outlineVariant,
                ),
                padding: const EdgeInsets.all(9),
              ),
              tooltip: isApplied ? 'Applied' : 'Mark as Applied',
            ),
            if (job != null && job.url.isNotEmpty) ...[
              const SizedBox(width: 6),
              IconButton.outlined(
                onPressed: () => _openJobUrl(job.url),
                icon: const Icon(
                  Icons.open_in_new,
                  color: AppColors.primary,
                  size: 19,
                ),
                style: IconButton.styleFrom(
                  side: const BorderSide(color: AppColors.outlineVariant),
                  padding: const EdgeInsets.all(9),
                ),
                tooltip: 'Open ATS Job Listing',
              ),
            ],
            const SizedBox(width: 8),
            OutlinedButton.icon(
              onPressed: _isGeneratingCoverLetter ? null : _handleGenerateCoverLetter,
              icon: _isGeneratingCoverLetter
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2, color: AppColors.primary),
                    )
                  : const Icon(Icons.auto_awesome, size: 15, color: AppColors.primary),
              label: Text(_isGeneratingCoverLetter ? 'Generating...' : 'AI Letter'),
              style: OutlinedButton.styleFrom(
                foregroundColor: AppColors.primary,
                side: const BorderSide(color: AppColors.outlineVariant),
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 12),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: ElevatedButton.icon(
                onPressed: _isTailoring ? null : _handleTailorResume,
                icon: _isTailoring
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                      )
                    : const Icon(Icons.auto_awesome, size: 15, color: Colors.white),
                label: Text(_isTailoring ? 'Tailoring...' : 'AI Resume'),
                style: ElevatedButton.styleFrom(
                  backgroundColor: AppColors.primary,
                  foregroundColor: Colors.white,
                  padding: const EdgeInsets.symmetric(vertical: 12),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}