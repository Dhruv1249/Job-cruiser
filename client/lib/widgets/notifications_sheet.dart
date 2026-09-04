import 'package:flutter/material.dart';
import '../details.dart' show CompanyDetailsPage;
import '../main.dart' show AppColors;
import '../models/job.dart';
import '../services/api_service.dart';

/// Shows a bottom sheet displaying user notifications for background operations.
Future<void> showNotificationsSheet(
  BuildContext context, {
  void Function(MatchedJob job)? onSelectJob,
}) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (sheetContext) => NotificationsSheet(onSelectJob: onSelectJob),
  );
}

/// NotificationsSheet displays recent system alerts, tailoring updates, and match progress.
class NotificationsSheet extends StatefulWidget {
  final void Function(MatchedJob job)? onSelectJob;
  final List<Map<String, dynamic>>? initialNotifications;

  const NotificationsSheet({super.key, this.onSelectJob, this.initialNotifications});

  @override
  State<NotificationsSheet> createState() => _NotificationsSheetState();
}

class _NotificationsSheetState extends State<NotificationsSheet> {
  final ApiService _apiService = ApiService();
  bool _isLoading = true;
  String? _openingJobId;
  List<Map<String, dynamic>> _notifications = [];

  @override
  void initState() {
    super.initState();
    if (widget.initialNotifications != null) {
      _notifications = List.from(widget.initialNotifications!);
      _isLoading = false;
    } else {
      _loadNotifications();
    }
  }

  Future<void> _loadNotifications() async {
    setState(() => _isLoading = true);
    final items = await _apiService.fetchNotifications();
    if (!mounted) return;
    setState(() {
      _notifications = items;
      _isLoading = false;
    });
  }

  Future<void> _markAsRead(String id, int index) async {
    final success = await _apiService.markNotificationRead(id);
    if (success && mounted) {
      setState(() {
        _notifications[index]['is_read'] = true;
      });
    }
  }

  Future<void> _markAllAsRead() async {
    final success = await _apiService.markAllNotificationsRead();
    if (success && mounted) {
      setState(() {
        for (final item in _notifications) {
          item['is_read'] = true;
        }
      });
    }
  }

  Future<void> _handleNotificationTap(Map<String, dynamic> notification, int index) async {
    final id = notification['id']?.toString() ?? '';
    final isRead = notification['is_read'] == true;
    if (!isRead && id.isNotEmpty) {
      _markAsRead(id, index);
    }

    final jobId = notification['job_id']?.toString();
    if (jobId != null && jobId.isNotEmpty) {
      setState(() => _openingJobId = jobId);
      final scaffoldMessenger = ScaffoldMessenger.of(context);
      final navigator = Navigator.of(context);

      final job = await _apiService.fetchJobById(jobId);
      if (!mounted) return;
      setState(() => _openingJobId = null);

      if (job != null) {
        navigator.pop();
        if (widget.onSelectJob != null) {
          widget.onSelectJob!(job);
        } else {
          navigator.push(
            MaterialPageRoute(
              builder: (_) => CompanyDetailsPage(job: job),
            ),
          );
        }
      } else {
        scaffoldMessenger.showSnackBar(
          const SnackBar(content: Text('Could not open job details. Please try again.')),
        );
      }
    }
  }

  String _formatTimestamp(String? raw) {
    if (raw == null || raw.isEmpty) return '';
    try {
      final dateTime = DateTime.parse(raw).toLocal();
      final difference = DateTime.now().difference(dateTime);
      if (difference.inMinutes < 1) return 'Just now';
      if (difference.inMinutes < 60) return '${difference.inMinutes}m ago';
      if (difference.inHours < 24) return '${difference.inHours}h ago';
      return '${difference.inDays}d ago';
    } catch (_) {
      return '';
    }
  }

  (String summary, String reasoning) _extractSummaryAndReasoning(Map<String, dynamic> notification) {
    final rawReasoning = notification['reasoning']?.toString().trim() ?? '';
    final rawMessage = notification['message']?.toString().trim() ?? '';

    if (rawReasoning.isNotEmpty) {
      var cleanSummary = rawMessage;
      if (cleanSummary.contains('AI Reasoning:')) {
        cleanSummary = cleanSummary.split('AI Reasoning:').first.trim();
      }
      return (cleanSummary, rawReasoning);
    }

    if (rawMessage.contains('AI Reasoning:')) {
      final splitIndex = rawMessage.indexOf('AI Reasoning:');
      final summaryPart = rawMessage.substring(0, splitIndex).trim();
      final reasoningPart = rawMessage.substring(splitIndex + 'AI Reasoning:'.length).trim();
      return (summaryPart, reasoningPart);
    }

    return (rawMessage, '');
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasUnread = _notifications.any((item) => item['is_read'] == false);

    return Container(
      height: MediaQuery.of(context).size.height * 0.8,
      decoration: const BoxDecoration(
        color: AppColors.surface,
        borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
      ),
      child: Column(
        children: [
          Container(
            margin: const EdgeInsets.only(top: 12, bottom: 8),
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: AppColors.outlineVariant,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 8),
            child: Row(
              children: [
                const Icon(Icons.notifications_active_outlined, color: AppColors.primary, size: 22),
                const SizedBox(width: 10),
                Text(
                  'Notifications',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.bold,
                    color: AppColors.primary,
                  ),
                ),
                const Spacer(),
                if (hasUnread)
                  TextButton(
                    onPressed: _markAllAsRead,
                    child: const Text('Mark all read'),
                  ),
                IconButton(
                  icon: const Icon(Icons.close, size: 20),
                  onPressed: () => Navigator.of(context).pop(),
                ),
              ],
            ),
          ),
          const Divider(height: 1, color: AppColors.outlineVariant),
          Expanded(
            child: _isLoading
                ? const Center(child: CircularProgressIndicator())
                : _notifications.isEmpty
                    ? Center(
                        child: Column(
                          mainAxisAlignment: MainAxisAlignment.center,
                          children: const [
                            Icon(Icons.notifications_none, size: 48, color: AppColors.onSurfaceVariant),
                            SizedBox(height: 12),
                            Text(
                              'No notifications yet',
                              style: TextStyle(
                                fontSize: 16,
                                fontWeight: FontWeight.w600,
                                color: AppColors.onSurfaceVariant,
                              ),
                            ),
                            SizedBox(height: 6),
                            Text(
                              'When high matches and background tailoring finish, alerts will appear here.',
                              style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
                              textAlign: TextAlign.center,
                            ),
                          ],
                        ),
                      )
                    : RefreshIndicator(
                        onRefresh: _loadNotifications,
                        child: ListView.separated(
                          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                          itemCount: _notifications.length,
                          separatorBuilder: (context, index) => const SizedBox(height: 10),
                          itemBuilder: (context, index) {
                            final notification = _notifications[index];
                            final title = notification['title']?.toString() ?? '';
                            final isRead = notification['is_read'] == true;
                            final timeString = _formatTimestamp(notification['created_at']?.toString());
                            final parsedContent = _extractSummaryAndReasoning(notification);
                            final summaryText = parsedContent.$1;
                            final reasoningText = parsedContent.$2;
                            final jobId = notification['job_id']?.toString();
                            final isOpening = _openingJobId == jobId && jobId != null;

                            return Material(
                              color: Colors.transparent,
                              child: InkWell(
                                onTap: () => _handleNotificationTap(notification, index),
                                borderRadius: BorderRadius.circular(12),
                                child: Container(
                                  padding: const EdgeInsets.all(14),
                                  decoration: BoxDecoration(
                                    color: isRead
                                        ? AppColors.surfaceContainerLowest
                                        : AppColors.primary.withValues(alpha: 0.04),
                                    borderRadius: BorderRadius.circular(12),
                                    border: Border.all(
                                      color: isRead
                                          ? AppColors.outlineVariant.withValues(alpha: 0.6)
                                          : AppColors.primary.withValues(alpha: 0.3),
                                    ),
                                  ),
                                  child: Column(
                                    crossAxisAlignment: CrossAxisAlignment.start,
                                    children: [
                                      Row(
                                        crossAxisAlignment: CrossAxisAlignment.start,
                                        children: [
                                          Container(
                                            width: 34,
                                            height: 34,
                                            decoration: BoxDecoration(
                                              shape: BoxShape.circle,
                                              color: isRead
                                                  ? AppColors.surfaceContainer
                                                  : AppColors.primary.withValues(alpha: 0.12),
                                            ),
                                            child: Icon(
                                              title.toLowerCase().contains('failed')
                                                  ? Icons.error_outline
                                                  : (reasoningText.isNotEmpty
                                                      ? Icons.auto_awesome
                                                      : Icons.notifications_none),
                                              color: title.toLowerCase().contains('failed')
                                                  ? Colors.redAccent
                                                  : (reasoningText.isNotEmpty
                                                      ? AppColors.matchGreen
                                                      : AppColors.primary),
                                              size: 18,
                                            ),
                                          ),
                                          const SizedBox(width: 10),
                                          Expanded(
                                            child: Column(
                                              crossAxisAlignment: CrossAxisAlignment.start,
                                              children: [
                                                Text(
                                                  title,
                                                  style: TextStyle(
                                                    fontWeight: isRead ? FontWeight.w600 : FontWeight.bold,
                                                    fontSize: 14,
                                                    color: AppColors.primary,
                                                  ),
                                                ),
                                                if (timeString.isNotEmpty)
                                                  Padding(
                                                    padding: const EdgeInsets.only(top: 2),
                                                    child: Text(
                                                      timeString,
                                                      style: const TextStyle(
                                                        fontSize: 11,
                                                        color: AppColors.onSurfaceVariant,
                                                      ),
                                                    ),
                                                  ),
                                              ],
                                            ),
                                          ),
                                          if (!isRead)
                                            Container(
                                              width: 8,
                                              height: 8,
                                              decoration: const BoxDecoration(
                                                color: AppColors.matchGreen,
                                                shape: BoxShape.circle,
                                              ),
                                            ),
                                        ],
                                      ),
                                      if (summaryText.isNotEmpty) ...[
                                        const SizedBox(height: 8),
                                        Text(
                                          summaryText,
                                          style: const TextStyle(
                                            fontSize: 13,
                                            color: AppColors.onSurface,
                                            height: 1.35,
                                          ),
                                        ),
                                      ],
                                      if (reasoningText.isNotEmpty) ...[
                                        const SizedBox(height: 10),
                                        Container(
                                          width: double.infinity,
                                          padding: const EdgeInsets.all(12),
                                          decoration: BoxDecoration(
                                            color: AppColors.surfaceContainerLowest,
                                            borderRadius: BorderRadius.circular(8),
                                            border: Border.all(
                                              color: AppColors.outlineVariant.withValues(alpha: 0.7),
                                            ),
                                          ),
                                          child: Column(
                                            crossAxisAlignment: CrossAxisAlignment.start,
                                            children: [
                                              Row(
                                                children: const [
                                                  Icon(Icons.auto_awesome, size: 14, color: AppColors.matchGreen),
                                                  SizedBox(width: 6),
                                                  Text(
                                                    'AI Match Reasoning',
                                                    style: TextStyle(
                                                      fontSize: 12,
                                                      fontWeight: FontWeight.bold,
                                                      color: AppColors.primary,
                                                    ),
                                                  ),
                                                ],
                                              ),
                                              const SizedBox(height: 6),
                                              Text(
                                                reasoningText,
                                                style: const TextStyle(
                                                  fontSize: 12.5,
                                                  color: AppColors.onSurface,
                                                  height: 1.4,
                                                ),
                                              ),
                                            ],
                                          ),
                                        ),
                                      ],
                                      if (jobId != null && jobId.isNotEmpty) ...[
                                        const SizedBox(height: 10),
                                        Row(
                                          mainAxisAlignment: MainAxisAlignment.end,
                                          children: [
                                            if (isOpening)
                                              const SizedBox(
                                                width: 16,
                                                height: 16,
                                                child: CircularProgressIndicator(strokeWidth: 2),
                                              )
                                            else
                                              Row(
                                                mainAxisSize: MainAxisSize.min,
                                                children: const [
                                                  Text(
                                                    'View Job Details',
                                                    style: TextStyle(
                                                      fontSize: 12,
                                                      fontWeight: FontWeight.w600,
                                                      color: AppColors.primary,
                                                    ),
                                                  ),
                                                  SizedBox(width: 4),
                                                  Icon(Icons.arrow_forward, size: 14, color: AppColors.primary),
                                                ],
                                              ),
                                          ],
                                        ),
                                      ],
                                    ],
                                  ),
                                ),
                              ),
                            );
                          },
                        ),
                      ),
          ),
        ],
      ),
    );
  }
}
