import 'package:flutter/material.dart';
import 'main.dart' show AppColors;
import 'models/job.dart';
import 'services/api_service.dart';

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
  String _currentStatus = 'bookmarked';

  MatchedJob? get _activeJob => widget.job;

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
      body: Stack(
        children: [
          SingleChildScrollView(
            padding: const EdgeInsets.only(
              left: 20,
              right: 20,
              top: 16,
              bottom: 120,
            ),
            child: Center(
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 600),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildHeroSection(job),
                    const SizedBox(height: 24),
                    _buildSectionHeader('Match Analysis'),
                    const SizedBox(height: 16),
                    _buildMatchAnalysis(job),
                    const SizedBox(height: 24),
                    _buildSectionHeader('Job Description & Requirements'),
                    const SizedBox(height: 16),
                    _buildJobDescription(job),
                  ],
                ),
              ),
            ),
          ),
          Positioned(
            left: 0,
            right: 0,
            bottom: 0,
            child: _buildBottomActionBar(),
          ),
        ],
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
                      const SizedBox(height: 6),
                      Text(
                        job.company,
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w600,
                          color: AppColors.onSurfaceVariant,
                        ),
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
                      'Status: ${_currentStatus.toUpperCase()}',
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
                if (job.source.isNotEmpty)
                  _buildInfoBadge(Icons.source_outlined, 'Source: ${job.source}'),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildInfoBadge(IconData icon, String text) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: [
        Icon(icon, size: 16, color: AppColors.onSurfaceVariant),
        const SizedBox(width: 4),
        Text(
          text,
          style: const TextStyle(
            fontSize: 13,
            fontWeight: FontWeight.w500,
            color: AppColors.onSurfaceVariant,
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
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Text(
        job.summary.isNotEmpty
            ? job.summary
            : 'Full job description available on original posting site.',
        style: const TextStyle(
          fontSize: 14,
          height: 1.6,
          color: AppColors.onSurface,
        ),
      ),
    );
  }

  Widget _buildBottomActionBar() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        border: const Border(
          top: BorderSide(color: AppColors.outlineVariant, width: 1.0),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.08),
            blurRadius: 16,
            offset: const Offset(0, -4),
          )
        ],
      ),
      child: Row(
        children: [
          Expanded(
            child: OutlinedButton.icon(
              onPressed: _isSaving
                  ? null
                  : () => _handleSaveStatus('bookmarked'),
              icon: const Icon(Icons.bookmark_border, size: 18),
              label: const Text('Save Job'),
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
              onPressed: _isSaving
                  ? null
                  : () => _handleSaveStatus('applied'),
              icon: const Icon(Icons.send, size: 18),
              label: const Text('Track as Applied'),
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                foregroundColor: Colors.white,
                padding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
        ],
      ),
    );
  }
}