import 'package:flutter/material.dart';
import 'main.dart' show AppColors;
import 'models/application.dart';
import 'services/api_service.dart';

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

  Future<void> _updateStatus(JobApplication app, String newStatus) async {
    final success = await _apiService.updateApplicationStatus(
      app.applicationId,
      newStatus,
    );

    if (!mounted) return;

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Updated status to ${app.statusDisplayLabel}'),
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
    return Scaffold(
      appBar: _buildAppBar(),
      body: RefreshIndicator(
        onRefresh: _loadApplications,
        color: AppColors.primary,
        child: Column(
          children: [
            _buildFilterChips(),
            Expanded(
              child: _isLoading
                  ? const Center(child: CircularProgressIndicator())
                  : _filteredApplications.isEmpty
                      ? _buildEmptyState()
                      : ListView.builder(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 20,
                            vertical: 12,
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
    );
  }

  PreferredSizeWidget _buildAppBar() {
    return AppBar(
      backgroundColor: AppColors.surface,
      elevation: 0,
      scrolledUnderElevation: 0,
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1.0),
        child: Container(
          color: AppColors.outlineVariant,
          height: 1.0,
        ),
      ),
      title: Row(
        children: [
          Container(
            width: 32,
            height: 32,
            decoration: const BoxDecoration(
              shape: BoxShape.circle,
              color: AppColors.surfaceContainerHigh,
            ),
            child: const Icon(
              Icons.work_history_outlined,
              size: 18,
              color: AppColors.primary,
            ),
          ),
          const SizedBox(width: 12),
          const Text(
            'Application Tracker',
            style: TextStyle(
              fontSize: 20,
              fontWeight: FontWeight.bold,
              color: AppColors.primary,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildFilterChips() {
    return Container(
      color: AppColors.surface,
      height: 54,
      child: ListView.builder(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
        itemCount: _statusFilters.length,
        itemBuilder: (context, index) {
          final filter = _statusFilters[index];
          final isSelected = filter == _selectedFilter;
          final label = filter == 'all'
              ? 'All (${_applications.length})'
              : '${_formatFilterLabel(filter)} (${_applications.where((a) => a.status.toLowerCase() == filter).length})';

          return Padding(
            padding: const EdgeInsets.only(right: 8),
            child: ChoiceChip(
              label: Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w600,
                  color: isSelected
                      ? AppColors.surfaceContainerLowest
                      : AppColors.onSurfaceVariant,
                ),
              ),
              selected: isSelected,
              selectedColor: AppColors.primary,
              backgroundColor: AppColors.surfaceContainerLowest,
              side: BorderSide(
                color: isSelected ? AppColors.primary : AppColors.outlineVariant,
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
            const SizedBox(height: 60),
            const Icon(
              Icons.assignment_outlined,
              size: 64,
              color: AppColors.outline,
            ),
            const SizedBox(height: 16),
            const Text(
              'No Applications Tracked Yet',
              style: TextStyle(
                fontSize: 18,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Jobs saved or applied for in the Inbox will appear here in your CRM pipeline.',
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

  Widget _buildApplicationCard(JobApplication app) {
    final statusColor = _getStatusColor(app.status);

    return Card(
      margin: const EdgeInsets.only(bottom: 12),
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: AppColors.outlineVariant),
      ),
      color: AppColors.surfaceContainerLowest,
      child: Padding(
        padding: const EdgeInsets.all(16),
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
                        app.title,
                        style: const TextStyle(
                          fontSize: 16,
                          fontWeight: FontWeight.w700,
                          color: AppColors.primary,
                        ),
                      ),
                      const SizedBox(height: 4),
                      Text(
                        app.companyName ?? app.companyId,
                        style: const TextStyle(
                          fontSize: 14,
                          fontWeight: FontWeight.w500,
                          color: AppColors.onSurfaceVariant,
                        ),
                      ),
                      if (app.location != null && app.location!.isNotEmpty) ...[
                        const SizedBox(height: 4),
                        Row(
                          children: [
                            const Icon(
                              Icons.location_on_outlined,
                              size: 14,
                              color: AppColors.outline,
                            ),
                            const SizedBox(width: 4),
                            Text(
                              app.location!,
                              style: const TextStyle(
                                fontSize: 12,
                                color: AppColors.outline,
                              ),
                            ),
                          ],
                        ),
                      ],
                    ],
                  ),
                ),
                PopupMenuButton<String>(
                  initialValue: app.status,
                  onSelected: (newStatus) => _updateStatus(app, newStatus),
                  child: Container(
                    padding: const EdgeInsets.symmetric(
                      horizontal: 10,
                      vertical: 6,
                    ),
                    decoration: BoxDecoration(
                      color: statusColor.withValues(alpha: 0.1),
                      borderRadius: BorderRadius.circular(16),
                      border: Border.all(color: statusColor),
                    ),
                    child: Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        Text(
                          app.statusDisplayLabel,
                          style: TextStyle(
                            fontSize: 12,
                            fontWeight: FontWeight.w600,
                            color: statusColor,
                          ),
                        ),
                        const SizedBox(width: 4),
                        Icon(
                          Icons.arrow_drop_down,
                          size: 16,
                          color: statusColor,
                        ),
                      ],
                    ),
                  ),
                  itemBuilder: (context) => [
                    const PopupMenuItem(
                      value: 'bookmarked',
                      child: Text('Bookmarked'),
                    ),
                    const PopupMenuItem(
                      value: 'applied',
                      child: Text('Applied'),
                    ),
                    const PopupMenuItem(
                      value: 'outreach_sent',
                      child: Text('Outreach Sent'),
                    ),
                    const PopupMenuItem(
                      value: 'interview',
                      child: Text('Interviewing'),
                    ),
                    const PopupMenuItem(
                      value: 'offer',
                      child: Text('Offer Received'),
                    ),
                    const PopupMenuItem(
                      value: 'rejected',
                      child: Text('Rejected'),
                    ),
                  ],
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Color _getStatusColor(String status) {
    switch (status.toLowerCase()) {
      case 'bookmarked':
        return AppColors.secondary;
      case 'applied':
        return AppColors.primary;
      case 'outreach_sent':
        return const Color(0xFF6366F1); // Indigo
      case 'interview':
      case 'interviewing':
        return const Color(0xFFF59E0B); // Amber
      case 'offer':
        return AppColors.successGreen;
      case 'rejected':
        return AppColors.error;
      default:
        return AppColors.outline;
    }
  }
}
