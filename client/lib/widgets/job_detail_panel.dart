import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../main.dart' show AppColors;
import '../models/job.dart';
import '../services/api_service.dart';
import '../preferences.dart' as preferences_page;
import 'company_logo_avatar.dart';
import 'job_description_renderer.dart';

/// Reusable panel component rendering full job details, match analysis, and interactive action controls.
class JobDetailPanel extends StatefulWidget {
  const JobDetailPanel({
    super.key,
    required this.job,
    this.onStatusChanged,
    this.onJobDismissed,
    this.showBackButton = false,
    this.onBackPressed,
  });

  final MatchedJob job;
  final Function(String status)? onStatusChanged;
  final Function(MatchedJob job)? onJobDismissed;
  final bool showBackButton;
  final VoidCallback? onBackPressed;

  @override
  State<JobDetailPanel> createState() => _JobDetailPanelState();
}

class _JobDetailPanelState extends State<JobDetailPanel> {
  final ApiService _apiService = ApiService();
  bool _isSaving = false;
  bool _isTailoring = false;
  late String _currentStatus;

  @override
  void initState() {
    super.initState();
    _currentStatus = widget.job.applicationStatus;
    if (widget.job.jobId.isNotEmpty) {
      _apiService.markJobAsViewed(widget.job.jobId);
    }
  }

  @override
  void didUpdateWidget(covariant JobDetailPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.job.jobId != widget.job.jobId) {
      _currentStatus = widget.job.applicationStatus;
      if (widget.job.jobId.isNotEmpty) {
        _apiService.markJobAsViewed(widget.job.jobId);
      }
    }
  }

  Future<void> _handleSaveStatus(String status) async {
    setState(() => _isSaving = true);
    final success = await _apiService.createApplication(widget.job.jobId, status);
    if (!mounted) return;

    setState(() {
      _isSaving = false;
      if (success) {
        _currentStatus = status;
      }
    });

    if (widget.onStatusChanged != null && success) {
      widget.onStatusChanged!(status);
    }

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
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: const Row(
          children: [
            Icon(Icons.visibility_off_outlined, color: AppColors.error),
            SizedBox(width: 8),
            Text('Hide this Job?'),
          ],
        ),
        content: Text(
          'Hide "${widget.job.title}" at "${widget.job.company}" from your feed? It will stay safely stored in the database, but won\'t appear in your matches.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text('Cancel'),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
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

    final success = await _apiService.dismissJob(widget.job.jobId);
    if (!mounted) return;

    if (success) {
      if (widget.onJobDismissed != null) {
        widget.onJobDismissed!(widget.job);
      }
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to hide job. Please try again.')),
      );
    }
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

  Future<void> _handleTailorApplication() async {
    setState(() => _isTailoring = true);
    try {
      final result = await _apiService.tailorApplicationAsync(jobId: widget.job.jobId);
      if (!mounted) return;

      if (result != null &&
          (result['status'] == 'processing' ||
              result['status_code'] == 202 ||
              result['message'] != null)) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: const Row(
              children: [
                Icon(Icons.check_circle, color: Colors.white, size: 20),
                SizedBox(width: 10),
                Expanded(
                  child: Text(
                    'Tailoring started in background! Resume & Cover Letter will compile in Open-Overleaf and notify you once ready.',
                  ),
                ),
              ],
            ),
            backgroundColor: AppColors.primary,
            duration: const Duration(seconds: 5),
          ),
        );
      } else {
        final errorMessage = result?['error'] as String? ?? '';
        final statusCode = result?['status_code'] as int? ?? 0;
        final isUnconfigured = result?['unconfigured'] == true || statusCode == 422;
        if (isUnconfigured ||
            errorMessage.toLowerCase().contains('overleaf') ||
            errorMessage.toLowerCase().contains('preferences') ||
            errorMessage.toLowerCase().contains('unconfigured')) {
          _showOverleafUnconfiguredDialog();
        } else {
          ScaffoldMessenger.of(context).showSnackBar(
            SnackBar(
              content: Text(
                errorMessage.isNotEmpty
                    ? errorMessage
                    : 'Tailoring failed to start. Please try again.',
              ),
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

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        if (widget.showBackButton) _buildHeaderBar(),
        Expanded(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 20),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 960),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildHeroCard(),
                    const SizedBox(height: 20),
                    _buildSectionTitle('Match Analysis'),
                    const SizedBox(height: 12),
                    _buildMatchAnalysisCard(),
                    const SizedBox(height: 20),
                    _buildSectionTitle('Job Description & Requirements'),
                    const SizedBox(height: 12),
                    _buildDescriptionCard(),
                    const SizedBox(height: 24),
                  ],
                ),
              ),
            ),
          ),
        ),
        _buildBottomActionBar(),
      ],
    );
  }

  Widget _buildHeaderBar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(
          bottom: BorderSide(color: AppColors.outlineVariant, width: 1),
        ),
      ),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.arrow_back, color: AppColors.primary),
            onPressed: widget.onBackPressed ?? () => Navigator.maybePop(context),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              widget.job.company,
              style: const TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          IconButton(
            icon: const Icon(Icons.visibility_off_outlined, color: AppColors.onSurfaceVariant),
            tooltip: 'Hide this Job',
            onPressed: _handleDismissJob,
          ),
        ],
      ),
    );
  }

  Widget _buildSectionTitle(String title) {
    return Text(
      title,
      style: const TextStyle(
        fontSize: 17,
        fontWeight: FontWeight.w700,
        color: AppColors.primary,
      ),
    );
  }

  Widget _buildHeroCard() {
    final isHighMatch = widget.job.matchScore >= 80;
    final isModerateMatch = widget.job.matchScore >= 60;

    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(color: AppColors.outlineVariant),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.04),
            blurRadius: 10,
            offset: const Offset(0, 3),
          )
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      widget.job.title,
                      style: const TextStyle(
                        fontSize: 22,
                        fontWeight: FontWeight.w800,
                        color: AppColors.primary,
                        letterSpacing: -0.3,
                      ),
                    ),
                    const SizedBox(height: 8),
                    Row(
                      children: [
                        CompanyLogoAvatar(
                          companyName: widget.job.company,
                          jobUrl: widget.job.url,
                          size: 26,
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: Text(
                            widget.job.company,
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
              const SizedBox(width: 12),
              Column(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                    decoration: BoxDecoration(
                      color: isHighMatch
                          ? AppColors.matchGreen
                          : isModerateMatch
                              ? AppColors.primaryContainer
                              : AppColors.surfaceContainerHigh,
                      borderRadius: BorderRadius.circular(16),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Icon(
                          Icons.bolt,
                          size: 14,
                          color: (isHighMatch || isModerateMatch)
                              ? Colors.white
                              : AppColors.outline,
                        ),
                        const SizedBox(width: 4),
                        Text(
                          widget.job.matchScore > 0
                              ? '${widget.job.matchScore}% Match'
                              : 'Unmatched',
                          style: TextStyle(
                            color: (isHighMatch || isModerateMatch)
                                ? Colors.white
                                : AppColors.outline,
                            fontSize: 12,
                            fontWeight: FontWeight.w700,
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
            spacing: 14,
            runSpacing: 8,
            children: [
              _buildBadge(Icons.location_on_outlined, widget.job.location),
              if (widget.job.isRemote)
                _buildBadge(Icons.wifi, 'Remote Position'),
              if (widget.job.seniority.isNotEmpty)
                _buildBadge(Icons.workspace_premium_outlined, widget.job.seniority),
              if (widget.job.salaryText.isNotEmpty)
                _buildBadge(Icons.attach_money, widget.job.salaryText),
              if (widget.job.scrapedAgoText.isNotEmpty)
                _buildBadge(Icons.schedule_outlined, 'Scraped ${widget.job.scrapedAgoText}'),
              if (widget.job.source.isNotEmpty)
                _buildBadge(Icons.source_outlined, 'Source: ${widget.job.source}'),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildBadge(IconData icon, String text) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, size: 15, color: AppColors.onSurfaceVariant),
        const SizedBox(width: 4),
        Flexible(
          child: Text(
            text,
            maxLines: 1,
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

  Widget _buildMatchAnalysisCard() {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
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
            widget.job.matchReasoning.isNotEmpty
                ? widget.job.matchReasoning
                : 'Direct role compatibility with your profile preferences.',
            style: const TextStyle(
              fontSize: 14,
              height: 1.5,
              color: AppColors.onSurface,
            ),
          ),
          if (widget.job.techStack.isNotEmpty) ...[
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
              children: widget.job.techStack
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
                      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                    ),
                  )
                  .toList(),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildDescriptionCard() {
    final hasRawDesc = widget.job.rawDescription.trim().isNotEmpty;
    final hasSummary = widget.job.summary.trim().isNotEmpty;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(18),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
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
              widget.job.summary,
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
              content: widget.job.rawDescription,
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
            if (widget.job.url.isNotEmpty) ...[
              const SizedBox(height: 12),
              InkWell(
                onTap: () => _openJobUrl(widget.job.url),
                child: Text(
                  widget.job.url,
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

  Widget _buildBottomActionBar() {
    final isBookmarked = _currentStatus == 'bookmarked';
    final isApplied = _currentStatus == 'applied';

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        border: const Border(
          top: BorderSide(color: AppColors.outlineVariant, width: 1),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.04),
            blurRadius: 8,
            offset: const Offset(0, -2),
          )
        ],
      ),
      child: SafeArea(
        top: false,
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 960),
            child: Row(
              children: [
                IconButton.outlined(
                  onPressed: _isSaving
                      ? null
                      : () => _handleSaveStatus(isBookmarked ? 'unbookmarked' : 'bookmarked'),
                  icon: Icon(
                    isBookmarked ? Icons.bookmark : Icons.bookmark_border,
                    color: isBookmarked ? AppColors.matchGreen : AppColors.primary,
                    size: 20,
                  ),
                  style: IconButton.styleFrom(
                    side: BorderSide(
                      color: isBookmarked ? AppColors.matchGreen : AppColors.outlineVariant,
                    ),
                    padding: const EdgeInsets.all(10),
                  ),
                  tooltip: isBookmarked ? 'Saved' : 'Save Job',
                ),
                const SizedBox(width: 8),
                IconButton.outlined(
                  onPressed: _isSaving
                      ? null
                      : () => _handleSaveStatus(isApplied ? 'not_applied' : 'applied'),
                  icon: Icon(
                    isApplied ? Icons.check_circle : Icons.check_circle_outline,
                    color: isApplied ? AppColors.matchGreen : AppColors.primary,
                    size: 20,
                  ),
                  style: IconButton.styleFrom(
                    side: BorderSide(
                      color: isApplied ? AppColors.matchGreen : AppColors.outlineVariant,
                    ),
                    padding: const EdgeInsets.all(10),
                  ),
                  tooltip: isApplied ? 'Applied' : 'Mark as Applied',
                ),
                if (widget.job.url.isNotEmpty) ...[
                  const SizedBox(width: 8),
                  IconButton.outlined(
                    onPressed: () => _openJobUrl(widget.job.url),
                    icon: const Icon(
                      Icons.open_in_new,
                      color: AppColors.primary,
                      size: 19,
                    ),
                    style: IconButton.styleFrom(
                      side: const BorderSide(color: AppColors.outlineVariant),
                      padding: const EdgeInsets.all(10),
                    ),
                    tooltip: 'Open ATS Job Listing',
                  ),
                ],
                const SizedBox(width: 8),
                IconButton.outlined(
                  onPressed: _handleDismissJob,
                  icon: const Icon(
                    Icons.visibility_off_outlined,
                    color: AppColors.error,
                    size: 19,
                  ),
                  style: IconButton.styleFrom(
                    side: const BorderSide(color: AppColors.outlineVariant),
                    padding: const EdgeInsets.all(10),
                  ),
                  tooltip: 'Hide Job',
                ),
                const Spacer(),
                Flexible(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(minWidth: 160, maxWidth: 280),
                    child: ElevatedButton.icon(
                      onPressed: _isTailoring ? null : _handleTailorApplication,
                      icon: _isTailoring
                          ? const SizedBox(
                              width: 15,
                              height: 15,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                            )
                          : const Icon(Icons.auto_awesome, size: 16, color: Colors.white),
                      label: Text(
                        _isTailoring ? 'Starting Tailoring...' : 'Tailor Application',
                        style: const TextStyle(fontWeight: FontWeight.w700, fontSize: 13),
                        overflow: TextOverflow.ellipsis,
                      ),
                      style: ElevatedButton.styleFrom(
                        backgroundColor: AppColors.primary,
                        foregroundColor: Colors.white,
                        padding: const EdgeInsets.symmetric(vertical: 12, horizontal: 16),
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
