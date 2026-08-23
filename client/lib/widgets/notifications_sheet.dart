import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../services/api_service.dart';

/// Shows a bottom sheet displaying user notifications for background operations.
Future<void> showNotificationsSheet(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (sheetContext) => const NotificationsSheet(),
  );
}

/// NotificationsSheet displays recent system alerts, tailoring updates, and match progress.
class NotificationsSheet extends StatefulWidget {
  const NotificationsSheet({super.key});

  @override
  State<NotificationsSheet> createState() => _NotificationsSheetState();
}

class _NotificationsSheetState extends State<NotificationsSheet> {
  final ApiService _apiService = ApiService();
  bool _isLoading = true;
  List<Map<String, dynamic>> _notifications = [];

  @override
  void initState() {
    super.initState();
    _loadNotifications();
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

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final hasUnread = _notifications.any((item) => item['is_read'] == false);

    return Container(
      height: MediaQuery.of(context).size.height * 0.75,
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
                              'When background tailoring finishes, alerts will appear here.',
                              style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
                              textAlign: TextAlign.center,
                            ),
                          ],
                        ),
                      )
                    : RefreshIndicator(
                        onRefresh: _loadNotifications,
                        child: ListView.separated(
                          padding: const EdgeInsets.symmetric(vertical: 8),
                          itemCount: _notifications.length,
                          separatorBuilder: (context, index) => const Divider(height: 1, color: AppColors.outlineVariant),
                          itemBuilder: (context, index) {
                            final notification = _notifications[index];
                            final id = notification['id']?.toString() ?? '';
                            final title = notification['title']?.toString() ?? '';
                            final message = notification['message']?.toString() ?? '';
                            final isRead = notification['is_read'] == true;
                            final timeString = _formatTimestamp(notification['created_at']?.toString());

                            return ListTile(
                              leading: Container(
                                width: 36,
                                height: 36,
                                decoration: BoxDecoration(
                                  shape: BoxShape.circle,
                                  color: isRead
                                      ? AppColors.surfaceContainer
                                      : AppColors.primary.withValues(alpha: 0.15),
                                ),
                                child: Icon(
                                  title.toLowerCase().contains('failed')
                                      ? Icons.error_outline
                                      : Icons.auto_awesome,
                                  color: title.toLowerCase().contains('failed')
                                      ? Colors.redAccent
                                      : AppColors.primary,
                                  size: 18,
                                ),
                              ),
                              title: Row(
                                children: [
                                  Expanded(
                                    child: Text(
                                      title,
                                      style: TextStyle(
                                        fontWeight: isRead ? FontWeight.w500 : FontWeight.bold,
                                        fontSize: 14,
                                        color: AppColors.primary,
                                      ),
                                    ),
                                  ),
                                  if (timeString.isNotEmpty)
                                    Text(
                                      timeString,
                                      style: const TextStyle(
                                        fontSize: 11,
                                        color: AppColors.onSurfaceVariant,
                                      ),
                                    ),
                                ],
                              ),
                              subtitle: Padding(
                                padding: const EdgeInsets.only(top: 4),
                                child: Text(
                                  message,
                                  style: const TextStyle(
                                    fontSize: 13,
                                    color: AppColors.onSurface,
                                    height: 1.3,
                                  ),
                                ),
                              ),
                              onTap: () {
                                if (!isRead && id.isNotEmpty) {
                                  _markAsRead(id, index);
                                }
                              },
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
