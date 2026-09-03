import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import 'main.dart' show AppColors;
import 'models/application.dart';
import 'models/job.dart';
import 'services/api_service.dart';
import 'widgets/company_logo_avatar.dart';
import 'details.dart' as details_page;

/// Screen representing the Application Tracker (CRM Pipeline).
class ApplicationTrackerPage extends StatefulWidget {
  const ApplicationTrackerPage({super.key, this.onSelectJobId});

  final Function(String jobId)? onSelectJobId;

  @override
  State<ApplicationTrackerPage> createState() => _ApplicationTrackerPageState();
}

class _ApplicationTrackerPageState extends State<ApplicationTrackerPage> {
  final ApiService _apiService = ApiService();
  List<JobApplication> _applications = [];
  bool _isLoading = true;
  String _selectedFilter = 'all';

  final List<String> _statusFilters = [
    'all',
    'bookmarked',
    'applied',
    'outreach_sent',
    'interview',
    'offer',
    'rejected',
  ];

  @override
  void initState() {
    super.initState();
    _loadApplications();
  }

  Future<void> _loadApplications() async {
    setState(() {
      _isLoading = true;
    });

    final apps = await _apiService.fetchApplications();

    if (!mounted) return;

    setState(() {
      _applications = apps;
      _isLoading = false;
    });
  }

  Future<void> _openJobDetails(JobApplication app) async {
    if (widget.onSelectJobId != null) {
      widget.onSelectJobId!(app.jobId);
    }

    final fullJob = await _apiService.fetchJobById(app.jobId);
    if (!mounted) return;

    final jobToOpen = fullJob ??
        MatchedJob(
          jobId: app.jobId,
          title: app.title,
          company: app.companyName ?? 'Company',
          location: app.location ?? '',
          isRemote: app.isRemote,
          source: 'CRM Tracker',
          url: app.url ?? '',
          postedDate: app.appliedAt ?? '',
          scrapedAt: '',
          seniority: app.seniority ?? '',
          summary: '',
          rawDescription: '',
          matchScore: app.matchScore ?? 0,
          matchReasoning: '',
          techStack: const [],
          isMatched: (app.matchScore ?? 0) >= 60,
          salaryMin: null,
          salaryMax: null,
          currency: 'USD',
          isViewed: true,
          applicationStatus: app.status,
          isNew: false,
        );

    Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => details_page.CompanyDetailsPage(job: jobToOpen),
      ),
    ).then((_) => _loadApplications());
  }

  Future<void> _confirmDeleteApplication(JobApplication app) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: const Row(
          children: [
            Icon(Icons.bookmark_remove_outlined, color: AppColors.error),
            SizedBox(width: 8),
            Text('Remove from Tracker?'),
          ],
        ),
        content: Text(
          'Remove "${app.title}" at "${app.companyName ?? 'this company'}" from your pipeline? It will still be visible in your matched feed.',
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
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            child: const Text('Remove'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    final success = await _apiService.deleteApplication(app.applicationId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Removed "${app.title}" from tracker'),
          duration: const Duration(seconds: 2),
        ),
      );
      _loadApplications();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to remove application. Please try again.'),
        ),
      );
    }
  }

  Future<void> _confirmDismissJob(JobApplication app) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
        title: const Row(
          children: [
            Icon(Icons.visibility_off_outlined, color: AppColors.error),
            SizedBox(width: 8),
            Text('Hide Job Completely?'),
          ],
        ),
        content: Text(
          'Hide "${app.title}" at "${app.companyName ?? 'this company'}" permanently from your account?\n\nThe job persists on the database, but will no longer appear in your tracker or feed.',
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
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            child: const Text('Hide Job'),
          ),
        ],
      ),
    );

    if (confirmed != true || !mounted) return;

    final success = await _apiService.dismissJob(app.jobId);
    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Hidden "${app.title}" from your account'),
          action: SnackBarAction(
            label: 'Undo',
            textColor: Colors.amber,
            onPressed: () async {
              await _apiService.undismissJob(app.jobId);
              _loadApplications();
            },
          ),
        ),
      );
      _loadApplications();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to hide job. Please try again.'),
        ),
      );
    }
  }

  Future<void> _updateStatus(JobApplication app, String newStatus) async {
    final success = await _apiService.updateApplicationStatus(
      app.applicationId,
      newStatus,
    );

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Moved to ${app.statusDisplayLabel}'),
          duration: const Duration(seconds: 2),
        ),
      );
      _loadApplications();
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text('Failed to update status. Please try again.'),
        ),
      );
    }
  }

  List<JobApplication> get _filteredApplications {
    if (_selectedFilter == 'all') {
      return _applications;
    }
    return _applications
        .where((app) => app.status.toLowerCase() == _selectedFilter.toLowerCase())
        .toList();
  }

  @override
  Widget build(BuildContext context) {
    final isDesktop = MediaQuery.of(context).size.width >= 960;
    return Scaffold(
      backgroundColor: AppColors.background,
      appBar: isDesktop ? null : _buildAppBar(),
      body: RefreshIndicator(
        onRefresh: _loadApplications,
        color: AppColors.primary,
        child: Center(
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 960),
            child: Column(
              children: [
                _buildPipelineMetricsBar(),
                _buildFilterChips(),
                Expanded(
                  child: _isLoading
                      ? const Center(child: CircularProgressIndicator())
                      : _filteredApplications.isEmpty
                          ? _buildEmptyState()
                          : ListView.builder(
                              padding: const EdgeInsets.only(
                                left: 16,
                                right: 16,
                                top: 12,
                                bottom: 24,
                              ),
                              itemCount: _filteredApplications.length,
                              itemBuilder: (context, index) {
                                final app = _filteredApplications[index];
                                return _buildApplicationCard(app);
                              },
                            ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar() {
    return AppBar(
      backgroundColor: AppColors.surfaceContainerLowest,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1.0),
        child: Container(
          color: AppColors.outlineVariant.withValues(alpha: 0.3),
          height: 1.0,
        ),
      ),
      title: Row(
        children: [
          Container(
            width: 34,
            height: 34,
            decoration: BoxDecoration(
              borderRadius: BorderRadius.circular(10),
              color: AppColors.primary.withValues(alpha: 0.06),
            ),
            child: const Icon(
              Icons.work_history_outlined,
              size: 20,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(width: 12),
          const Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                'Application Tracker',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: AppColors.primary,
                  letterSpacing: -0.3,
                ),
              ),
              Text(
                'CRM Pipeline & Outreach',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w500,
                  color: AppColors.outline,
                ),
              ),
            ],
          ),
        ],
      ),
      actions: [
        IconButton(
          icon: const Icon(Icons.refresh, size: 20, color: AppColors.onSurfaceVariant),
          tooltip: 'Refresh Pipeline',
          onPressed: _loadApplications,
        ),
      ],
    );
  }

  Widget _buildPipelineMetricsBar() {
    final totalCount = _applications.length;
    final appliedCount = _applications.where((a) => a.status.toLowerCase() == 'applied' || a.status.toLowerCase() == 'outreach_sent').length;
    final interviewCount = _applications.where((a) => a.status.toLowerCase() == 'interview' || a.status.toLowerCase() == 'interviewing').length;
    final offerCount = _applications.where((a) => a.status.toLowerCase() == 'offer').length;

    return Container(
      color: AppColors.surfaceContainerLowest,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      child: Row(
        children: [
          Expanded(
            child: _buildMetricTile(
              label: 'Total',
              count: totalCount,
              color: AppColors.primary,
              icon: Icons.list_alt,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildMetricTile(
              label: 'Applied',
              count: appliedCount,
              color: const Color(0xFF2563EB),
              icon: Icons.send_outlined,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildMetricTile(
              label: 'Interview',
              count: interviewCount,
              color: const Color(0xFFD97706),
              icon: Icons.record_voice_over_outlined,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: _buildMetricTile(
              label: 'Offers',
              count: offerCount,
              color: AppColors.successGreen,
              icon: Icons.emoji_events_outlined,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMetricTile({
    required String label,
    required int count,
    required Color color,
    required IconData icon,
  }) {
    return Container(
      padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 8),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.05),
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: color.withValues(alpha: 0.15)),
      ),
      child: Column(
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              Icon(icon, size: 14, color: color),
              const SizedBox(width: 4),
              Text(
                '$count',
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w800,
                  color: color,
                ),
              ),
            ],
          ),
          const SizedBox(height: 2),
          Text(
            label,
            style: TextStyle(
              fontSize: 10,
              fontWeight: FontWeight.w600,
              color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterChips() {
    return Container(
      color: AppColors.surfaceContainerLowest,
      height: 48,
      child: ListView.builder(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 6),
        itemCount: _statusFilters.length,
        itemBuilder: (context, index) {
          final filter = _statusFilters[index];
          final isSelected = filter == _selectedFilter;
          final count = filter == 'all'
              ? _applications.length
              : _applications.where((a) => a.status.toLowerCase() == filter).length;
          final label = filter == 'all'
              ? 'All ($count)'
              : '${_formatFilterLabel(filter)} ($count)';
          final stageColor = _getStatusColor(filter);

          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: ChoiceChip(
              avatar: filter != 'all'
                  ? Container(
                      width: 8,
                      height: 8,
                      decoration: BoxDecoration(
                        shape: BoxShape.circle,
                        color: isSelected ? Colors.white : stageColor,
                      ),
                    )
                  : null,
              label: Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: isSelected
                      ? Colors.white
                      : AppColors.onSurfaceVariant,
                ),
              ),
              selected: isSelected,
              selectedColor: AppColors.primary,
              backgroundColor: AppColors.background,
              shape: RoundedRectangleBorder(
                borderRadius: BorderRadius.circular(20),
                side: BorderSide(
                  color: isSelected ? AppColors.primary : AppColors.outlineVariant.withValues(alpha: 0.4),
                ),
              ),
              onSelected: (selected) {
                if (selected) {
                  setState(() {
                    _selectedFilter = filter;
                  });
                }
              },
            ),
          );
        },
      ),
    );
  }

  String _formatFilterLabel(String filter) {
    switch (filter) {
      case 'bookmarked':
        return 'Saved';
      case 'applied':
        return 'Applied';
      case 'outreach_sent':
        return 'Outreach';
      case 'interview':
        return 'Interview';
      case 'offer':
        return 'Offer';
      case 'rejected':
        return 'Rejected';
      default:
        return filter;
    }
  }

  Widget _buildEmptyState() {
    return SingleChildScrollView(
      physics: const AlwaysScrollableScrollPhysics(),
      child: Container(
        padding: const EdgeInsets.all(40),
        alignment: Alignment.center,
        child: Column(
          mainAxisAlignment: MainAxisAlignment.center,
          children: [
            const SizedBox(height: 48),
            Container(
              width: 80,
              height: 80,
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                color: AppColors.primary.withValues(alpha: 0.05),
              ),
              child: const Icon(
                Icons.work_outline,
                size: 40,
                color: AppColors.outline,
              ),
            ),
            const SizedBox(height: 18),
            Text(
              _selectedFilter == 'all'
                  ? 'No Applications in Pipeline'
                  : 'No jobs in "${_formatFilterLabel(_selectedFilter)}"',
              style: const TextStyle(
                fontSize: 17,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Save or apply to positions from your Matched Jobs feed to track progress through your CRM pipeline.',
              textAlign: TextAlign.center,
              style: TextStyle(
                fontSize: 13,
                height: 1.4,
                color: AppColors.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildApplicationCard(JobApplication app) {
    final statusColor = _getStatusColor(app.status);

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(14),
        border: Border.all(
          color: AppColors.outlineVariant.withValues(alpha: 0.4),
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.03),
            blurRadius: 8,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: ClipRRect(
        borderRadius: BorderRadius.circular(14),
        child: InkWell(
          onTap: () => _openJobDetails(app),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                height: 3,
                color: statusColor,
              ),
              Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        CompanyLogoAvatar(
                          companyName: app.companyName ?? app.companyId,
                          jobUrl: app.url ?? '',
                          size: 38,
                        ),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                app.title,
                                style: const TextStyle(
                                  fontSize: 15,
                                  fontWeight: FontWeight.w700,
                                  color: AppColors.primary,
                                  height: 1.25,
                                ),
                                maxLines: 2,
                                overflow: TextOverflow.ellipsis,
                              ),
                              const SizedBox(height: 4),
                              Text(
                                app.companyName ?? app.companyId,
                                style: const TextStyle(
                                  fontSize: 13,
                                  fontWeight: FontWeight.w600,
                                  color: AppColors.secondary,
                                ),
                              ),
                            ],
                          ),
                        ),
                        const SizedBox(width: 8),
                        _buildStatusDropdown(app, statusColor),
                        const SizedBox(width: 4),
                        _buildMoreMenu(app),
                      ],
                    ),
                    const SizedBox(height: 10),
                    if (app.location != null && app.location!.isNotEmpty) ...[
                      Row(
                        children: [
                          Icon(
                            app.isRemote ? Icons.wifi : Icons.location_on_outlined,
                            size: 13,
                            color: AppColors.outline,
                          ),
                          const SizedBox(width: 4),
                          Flexible(
                            child: Text(
                              app.location!,
                              style: const TextStyle(
                                fontSize: 12,
                                fontWeight: FontWeight.w500,
                                color: AppColors.outline,
                              ),
                              maxLines: 1,
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                        ],
                      ),
                      const SizedBox(height: 10),
                    ],
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        Wrap(
                          spacing: 6,
                          runSpacing: 4,
                          children: [
                            if (app.matchScore != null && app.matchScore! > 0)
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                                decoration: BoxDecoration(
                                  color: AppColors.matchGreen.withValues(alpha: 0.1),
                                  borderRadius: BorderRadius.circular(6),
                                  border: Border.all(color: AppColors.matchGreen.withValues(alpha: 0.3)),
                                ),
                                child: Row(
                                  mainAxisSize: MainAxisSize.min,
                                  children: [
                                    const Icon(Icons.bolt, size: 12, color: AppColors.matchGreen),
                                    const SizedBox(width: 2),
                                    Text(
                                      '${app.matchScore}% Match',
                                      style: const TextStyle(
                                        fontSize: 11,
                                        fontWeight: FontWeight.w700,
                                        color: AppColors.matchGreen,
                                      ),
                                    ),
                                  ],
                                ),
                              ),
                            if (app.isRemote)
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                                decoration: BoxDecoration(
                                  color: AppColors.secondaryContainer,
                                  borderRadius: BorderRadius.circular(6),
                                ),
                                child: const Text(
                                  'Remote',
                                  style: TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    color: AppColors.onSecondaryContainer,
                                  ),
                                ),
                              ),
                            if (app.seniority != null && app.seniority!.isNotEmpty)
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 7, vertical: 3),
                                decoration: BoxDecoration(
                                  color: AppColors.surfaceContainerHigh,
                                  borderRadius: BorderRadius.circular(6),
                                ),
                                child: Text(
                                  app.seniority!,
                                  style: const TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w600,
                                    color: AppColors.onSurfaceVariant,
                                  ),
                                ),
                              ),
                          ],
                        ),
                        if (app.url != null && app.url!.isNotEmpty)
                          InkWell(
                            onTap: () async {
                              final uri = Uri.tryParse(app.url!);
                              if (uri != null && await canLaunchUrl(uri)) {
                                await launchUrl(uri, mode: LaunchMode.externalApplication);
                              }
                            },
                            borderRadius: BorderRadius.circular(6),
                            child: const Padding(
                              padding: EdgeInsets.symmetric(horizontal: 6, vertical: 3),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: [
                                  Text(
                                    'View Post',
                                    style: TextStyle(
                                      fontSize: 12,
                                      fontWeight: FontWeight.w600,
                                      color: AppColors.primary,
                                    ),
                                  ),
                                  SizedBox(width: 4),
                                  Icon(Icons.open_in_new, size: 13, color: AppColors.primary),
                                ],
                              ),
                            ),
                          ),
                      ],
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildStatusDropdown(JobApplication app, Color statusColor) {
    return PopupMenuButton<String>(
      initialValue: app.status,
      onSelected: (newStatus) => _updateStatus(app, newStatus),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 5),
        decoration: BoxDecoration(
          color: statusColor.withValues(alpha: 0.1),
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: statusColor.withValues(alpha: 0.3)),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              width: 6,
              height: 6,
              decoration: BoxDecoration(shape: BoxShape.circle, color: statusColor),
            ),
            const SizedBox(width: 6),
            Text(
              app.statusDisplayLabel,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w700,
                color: statusColor,
              ),
            ),
            const SizedBox(width: 2),
            Icon(Icons.keyboard_arrow_down, size: 14, color: statusColor),
          ],
        ),
      ),
      itemBuilder: (context) => [
        const PopupMenuItem(
          value: 'bookmarked',
          child: Row(
            children: [
              Icon(Icons.bookmark_outline, size: 18, color: Color(0xFFD97706)),
              SizedBox(width: 10),
              Text('Bookmarked'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'applied',
          child: Row(
            children: [
              Icon(Icons.send_outlined, size: 18, color: Color(0xFF2563EB)),
              SizedBox(width: 10),
              Text('Applied'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'outreach_sent',
          child: Row(
            children: [
              Icon(Icons.mail_outline, size: 18, color: Color(0xFF6366F1)),
              SizedBox(width: 10),
              Text('Outreach Sent'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'interview',
          child: Row(
            children: [
              Icon(Icons.record_voice_over_outlined, size: 18, color: Color(0xFFF59E0B)),
              SizedBox(width: 10),
              Text('Interviewing'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'offer',
          child: Row(
            children: [
              Icon(Icons.emoji_events_outlined, size: 18, color: AppColors.successGreen),
              SizedBox(width: 10),
              Text('Offer Received'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'rejected',
          child: Row(
            children: [
              Icon(Icons.cancel_outlined, size: 18, color: AppColors.error),
              SizedBox(width: 10),
              Text('Rejected'),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildMoreMenu(JobApplication app) {
    return PopupMenuButton<String>(
      icon: const Icon(Icons.more_vert, size: 18, color: AppColors.onSurfaceVariant),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (action) {
        if (action == 'details') {
          _openJobDetails(app);
        } else if (action == 'remove') {
          _confirmDeleteApplication(app);
        } else if (action == 'dismiss') {
          _confirmDismissJob(app);
        }
      },
      itemBuilder: (context) => [
        const PopupMenuItem(
          value: 'details',
          child: Row(
            children: [
              Icon(Icons.open_in_new, size: 16, color: AppColors.primary),
              SizedBox(width: 8),
              Text('Open Job Details'),
            ],
          ),
        ),
        const PopupMenuItem(
          value: 'remove',
          child: Row(
            children: [
              Icon(Icons.bookmark_remove_outlined, size: 16, color: AppColors.onSurfaceVariant),
              SizedBox(width: 8),
              Text('Remove from Tracker'),
            ],
          ),
        ),
        const PopupMenuDivider(),
        const PopupMenuItem(
          value: 'dismiss',
          child: Row(
            children: [
              Icon(Icons.visibility_off_outlined, size: 16, color: AppColors.error),
              SizedBox(width: 8),
              Text(
                'Hide / Dismiss Job',
                style: TextStyle(color: AppColors.error),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Color _getStatusColor(String status) {
    switch (status.toLowerCase()) {
      case 'bookmarked':
        return const Color(0xFFD97706); // Amber
      case 'applied':
        return const Color(0xFF2563EB); // Royal Blue
      case 'outreach_sent':
        return const Color(0xFF6366F1); // Indigo
      case 'interview':
      case 'interviewing':
        return const Color(0xFFEA580C); // Warm Orange
      case 'offer':
        return AppColors.successGreen; // Emerald
      case 'rejected':
        return AppColors.error;
      default:
        return AppColors.outline;
    }
  }
}

