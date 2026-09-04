import 'package:flutter/material.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Displays scraper execution health metrics and an interactive history log of past scraper runs.
class ScraperRunHistoryCard extends StatefulWidget {
  const ScraperRunHistoryCard({
    super.key,
    required this.runHealth,
    required this.runs,
  });

  final ScraperRunHealthMetric runHealth;
  final List<ScraperRunLog> runs;

  @override
  State<ScraperRunHistoryCard> createState() => _ScraperRunHistoryCardState();
}

class _ScraperRunHistoryCardState extends State<ScraperRunHistoryCard> {
  int _maxVisibleRuns = 5;

  @override
  Widget build(BuildContext context) {
    final health = widget.runHealth;
    final runs = widget.runs;
    final visibleRuns = runs.take(_maxVisibleRuns).toList();

    return Material(
      color: AppColors.surfaceContainerLowest,
      borderRadius: BorderRadius.circular(10),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.02),
              blurRadius: 6,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    const Text(
                      'Scraper Execution Health & Run History',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        color: AppColors.primary,
                        letterSpacing: -0.3,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Monitoring Cloud Run job reliability, intake yields, and error logs',
                      style: TextStyle(
                        fontSize: 12,
                        color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
                      ),
                    ),
                  ],
                ),
              ),
              _buildSuccessRateBadge(health.successRatePct),
            ],
          ),
          const SizedBox(height: 16),
          LayoutBuilder(
            builder: (context, constraints) {
              final isCompact = constraints.maxWidth < 600;

              return Container(
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  color: AppColors.surfaceContainerLow,
                  borderRadius: BorderRadius.circular(8),
                ),
                child: isCompact
                    ? Column(
                        children: [
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceAround,
                            children: [
                              _buildMiniStat('Total Runs', '${health.totalRunsRecorded}'),
                              _buildMiniStat('Successful', '${health.successfulRuns}', color: AppColors.matchGreen),
                            ],
                          ),
                          const Divider(height: 16, color: AppColors.outlineVariant),
                          Row(
                            mainAxisAlignment: MainAxisAlignment.spaceAround,
                            children: [
                              _buildMiniStat('Failed', '${health.failedRuns}', color: health.failedRuns > 0 ? AppColors.error : AppColors.secondary),
                              _buildMiniStat('Avg Duration', _formatDuration(health.avgDurationSeconds)),
                            ],
                          ),
                        ],
                      )
                    : Row(
                        mainAxisAlignment: MainAxisAlignment.spaceAround,
                        children: [
                          _buildMiniStat('Total Runs', '${health.totalRunsRecorded}'),
                          _buildMiniStat('Successful', '${health.successfulRuns}', color: AppColors.matchGreen),
                          _buildMiniStat('Failed', '${health.failedRuns}', color: health.failedRuns > 0 ? AppColors.error : AppColors.secondary),
                          _buildMiniStat('Avg Duration', _formatDuration(health.avgDurationSeconds)),
                        ],
                      ),
              );
            },
          ),
          const SizedBox(height: 16),
          if (runs.isEmpty)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 20),
              child: Center(
                child: Text(
                  'No scraper run logs recorded yet.',
                  style: TextStyle(color: AppColors.onSurfaceVariant),
                ),
              ),
            )
          else
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: visibleRuns.length,
              separatorBuilder: (context, index) => const Divider(height: 12, color: AppColors.surfaceContainer),
              itemBuilder: (context, index) {
                final run = visibleRuns[index];
                final isFailed = run.status.toLowerCase() == 'failed' || run.status.toLowerCase() == 'error';
                final isRunning = run.status.toLowerCase() == 'running';

                return ExpansionTile(
                  tilePadding: EdgeInsets.zero,
                  childrenPadding: const EdgeInsets.only(bottom: 8),
                  collapsedIconColor: AppColors.onSurfaceVariant,
                  iconColor: AppColors.primary,
                  title: Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                        decoration: BoxDecoration(
                          color: isFailed
                              ? AppColors.error.withValues(alpha: 0.12)
                              : (isRunning ? Colors.amber.withValues(alpha: 0.15) : AppColors.matchGreen.withValues(alpha: 0.12)),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          run.status.toUpperCase(),
                          style: TextStyle(
                            fontSize: 10,
                            fontWeight: FontWeight.w800,
                            color: isFailed
                                ? AppColors.error
                                : (isRunning ? Colors.amber.shade900 : AppColors.matchGreen),
                          ),
                        ),
                      ),
                      const SizedBox(width: 10),
                      Expanded(
                        child: Text(
                          _formatTimestamp(run.startedAt),
                          style: const TextStyle(
                            fontSize: 13,
                            fontWeight: FontWeight.w600,
                            color: AppColors.primary,
                          ),
                        ),
                      ),
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                        decoration: BoxDecoration(
                          color: AppColors.surfaceContainerHigh,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          '+${run.jobsAdded} jobs',
                          style: const TextStyle(
                            fontSize: 11,
                            fontWeight: FontWeight.w700,
                            color: AppColors.primary,
                          ),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        _formatDuration(run.durationSeconds),
                        style: TextStyle(
                          fontSize: 12,
                          color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
                        ),
                      ),
                    ],
                  ),
                  children: [
                    if (isFailed && run.errorMessage.isNotEmpty)
                      Container(
                        width: double.infinity,
                        margin: const EdgeInsets.only(bottom: 8),
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: AppColors.error.withValues(alpha: 0.08),
                          borderRadius: BorderRadius.circular(6),
                          border: Border.all(color: AppColors.error.withValues(alpha: 0.3)),
                        ),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Icon(Icons.error_outline, size: 16, color: AppColors.error),
                            const SizedBox(width: 8),
                            Expanded(
                              child: Text(
                                run.errorMessage,
                                style: const TextStyle(
                                  fontSize: 12,
                                  color: AppColors.error,
                                  fontFamily: 'monospace',
                                ),
                              ),
                            ),
                          ],
                        ),
                      ),
                    if (run.sourcesList.isNotEmpty)
                      Padding(
                        padding: const EdgeInsets.only(top: 4),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text(
                              'Sources hit: ',
                              style: TextStyle(
                                fontSize: 11,
                                fontWeight: FontWeight.w600,
                                color: AppColors.secondary,
                              ),
                            ),
                            Expanded(
                              child: Wrap(
                                spacing: 6,
                                runSpacing: 4,
                                children: run.sourcesList.map((source) {
                                  return Container(
                                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                    decoration: BoxDecoration(
                                      color: AppColors.surfaceContainerHigh,
                                      borderRadius: BorderRadius.circular(4),
                                    ),
                                    child: Text(
                                      source,
                                      style: const TextStyle(
                                        fontSize: 10,
                                        fontWeight: FontWeight.w500,
                                        color: AppColors.primary,
                                      ),
                                    ),
                                  );
                                }).toList(),
                              ),
                            ),
                          ],
                        ),
                      ),
                  ],
                );
              },
            ),
          if (runs.length > 5) ...[
            const SizedBox(height: 8),
            Center(
              child: TextButton.icon(
                onPressed: () {
                  setState(() {
                    if (_maxVisibleRuns >= runs.length) {
                      _maxVisibleRuns = 5;
                    } else {
                      _maxVisibleRuns = runs.length;
                    }
                  });
                },
                icon: Icon(
                  _maxVisibleRuns >= runs.length ? Icons.keyboard_arrow_up : Icons.keyboard_arrow_down,
                  size: 18,
                  color: AppColors.secondary,
                ),
                label: Text(
                  _maxVisibleRuns >= runs.length ? 'Show Less' : 'View All ${runs.length} Runs',
                  style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    color: AppColors.secondary,
                  ),
                ),
              ),
            ),
          ],
        ],
      ),
    ),
  );
}

  Widget _buildSuccessRateBadge(double rate) {
    final color = rate >= 95.0
        ? AppColors.matchGreen
        : (rate >= 80.0 ? Colors.amber.shade700 : AppColors.error);

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(16),
        border: Border.all(color: color.withValues(alpha: 0.3)),
      ),
      child: Text(
        '${rate.toStringAsFixed(0)}% Success Rate',
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w700,
          color: color,
        ),
      ),
    );
  }

  Widget _buildMiniStat(String label, String value, {Color? color}) {
    return Column(
      children: [
        Text(
          value,
          style: TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.w800,
            color: color ?? AppColors.primary,
          ),
        ),
        const SizedBox(height: 2),
        Text(
          label,
          style: TextStyle(
            fontSize: 11,
            color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
          ),
        ),
      ],
    );
  }

  String _formatDuration(int seconds) {
    if (seconds <= 0) return '0s';
    if (seconds < 60) return '${seconds}s';
    final minutes = seconds ~/ 60;
    final remainingSeconds = seconds % 60;
    if (remainingSeconds == 0) return '${minutes}m';
    return '${minutes}m ${remainingSeconds}s';
  }

  String _formatTimestamp(String timestamp) {
    if (timestamp.isEmpty) return 'Recent';
    try {
      final parsed = DateTime.parse(timestamp).toLocal();
      final monthNames = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
      final month = monthNames[parsed.month - 1];
      final day = parsed.day;
      final hour = parsed.hour.toString().padLeft(2, '0');
      final minute = parsed.minute.toString().padLeft(2, '0');
      return '$month $day, $hour:$minute';
    } catch (_) {
      return timestamp;
    }
  }
}
