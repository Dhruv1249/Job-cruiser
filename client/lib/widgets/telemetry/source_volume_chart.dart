import 'package:flutter/material.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Renders a comparative volume breakdown of jobs ingested across all scraping sources.
class SourceVolumeChart extends StatefulWidget {
  const SourceVolumeChart({
    super.key,
    required this.sourcesVolume,
  });

  final List<SourceVolumeMetric> sourcesVolume;

  @override
  State<SourceVolumeChart> createState() => _SourceVolumeChartState();
}

class _SourceVolumeChartState extends State<SourceVolumeChart> {
  String _activeSortMode = 'volume';

  @override
  Widget build(BuildContext context) {
    if (widget.sourcesVolume.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
        ),
        child: const Center(
          child: Text(
            'No source volume metrics available.',
            textAlign: TextAlign.center,
            style: TextStyle(color: AppColors.onSurfaceVariant),
          ),
        ),
      );
    }

    final sortedList = List<SourceVolumeMetric>.from(widget.sourcesVolume);
    if (_activeSortMode == 'volume') {
      sortedList.sort((a, b) => b.totalJobs.compareTo(a.totalJobs));
    } else if (_activeSortMode == 'velocity') {
      sortedList.sort((a, b) => b.jobsLast24h.compareTo(a.jobsLast24h));
    } else if (_activeSortMode == 'remote') {
      sortedList.sort((a, b) => b.remoteJobs.compareTo(a.remoteJobs));
    }

    final maxVolume = sortedList.fold<int>(1, (prev, element) => element.totalJobs > prev ? element.totalJobs : prev);

    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
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
                      'Source Volume & Intake Velocity',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        color: AppColors.primary,
                        letterSpacing: -0.3,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Distribution of scraped listings and 24h intake rates',
                      style: TextStyle(
                        fontSize: 12,
                        color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
                      ),
                    ),
                  ],
                ),
              ),
              PopupMenuButton<String>(
                initialValue: _activeSortMode,
                onSelected: (mode) => setState(() => _activeSortMode = mode),
                itemBuilder: (context) => const [
                  PopupMenuItem(value: 'volume', child: Text('Sort by Total Jobs')),
                  PopupMenuItem(value: 'velocity', child: Text('Sort by +24h Additions')),
                  PopupMenuItem(value: 'remote', child: Text('Sort by Remote Jobs')),
                ],
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: AppColors.surfaceContainerLow,
                    borderRadius: BorderRadius.circular(6),
                    border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.6)),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      const Icon(Icons.sort, size: 14, color: AppColors.onSurfaceVariant),
                      const SizedBox(width: 4),
                      Text(
                        _sortModeLabel(_activeSortMode),
                        style: const TextStyle(
                          fontSize: 12,
                          fontWeight: FontWeight.w600,
                          color: AppColors.onSurfaceVariant,
                        ),
                      ),
                      const Icon(Icons.arrow_drop_down, size: 16, color: AppColors.onSurfaceVariant),
                    ],
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          ListView.separated(
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            itemCount: sortedList.length,
            separatorBuilder: (context, index) => const Divider(height: 16, color: AppColors.surfaceContainer),
            itemBuilder: (context, index) {
              final metric = sortedList[index];
              final proportion = maxVolume > 0 ? (metric.totalJobs / maxVolume).clamp(0.0, 1.0) : 0.0;
              final remotePct = metric.totalJobs > 0 ? (metric.remoteJobs * 100.0 / metric.totalJobs) : 0.0;

              return Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Expanded(
                        child: Row(
                          children: [
                            Text(
                              _formatSourceName(metric.source),
                              style: const TextStyle(
                                fontSize: 14,
                                fontWeight: FontWeight.w600,
                                color: AppColors.primary,
                              ),
                            ),
                            const SizedBox(width: 8),
                            if (metric.jobsLast24h > 0)
                              Container(
                                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                decoration: BoxDecoration(
                                  color: AppColors.matchGreen.withValues(alpha: 0.12),
                                  borderRadius: BorderRadius.circular(4),
                                ),
                                child: Text(
                                  '+${metric.jobsLast24h} 24h',
                                  style: const TextStyle(
                                    fontSize: 11,
                                    fontWeight: FontWeight.w700,
                                    color: AppColors.matchGreen,
                                  ),
                                ),
                              ),
                          ],
                        ),
                      ),
                      Text(
                        '${metric.totalJobs} jobs',
                        style: const TextStyle(
                          fontSize: 13,
                          fontWeight: FontWeight.w700,
                          color: AppColors.primary,
                        ),
                      ),
                      const SizedBox(width: 6),
                      Text(
                        '(${metric.sharePct.toStringAsFixed(1)}%)',
                        style: TextStyle(
                          fontSize: 12,
                          color: AppColors.onSurfaceVariant.withValues(alpha: 0.7),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 6),
                  ClipRRect(
                    borderRadius: BorderRadius.circular(3),
                    child: SizedBox(
                      height: 6,
                      child: LinearProgressIndicator(
                        value: proportion,
                        backgroundColor: AppColors.surfaceContainerHigh,
                        valueColor: const AlwaysStoppedAnimation<Color>(AppColors.primary),
                      ),
                    ),
                  ),
                  const SizedBox(height: 6),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        '${metric.remoteJobs} remote (${remotePct.toStringAsFixed(0)}%) · ${metric.onsiteJobs} on-site',
                        style: TextStyle(
                          fontSize: 11,
                          color: AppColors.onSurfaceVariant.withValues(alpha: 0.75),
                        ),
                      ),
                      if (metric.jobsLast7d > 0)
                        Text(
                          '${metric.jobsLast7d} in 7d',
                          style: TextStyle(
                            fontSize: 11,
                            fontWeight: FontWeight.w500,
                            color: AppColors.secondary.withValues(alpha: 0.9),
                          ),
                        ),
                    ],
                  ),
                ],
              );
            },
          ),
        ],
      ),
    );
  }

  String _sortModeLabel(String mode) {
    switch (mode) {
      case 'velocity':
        return '+24h Velocity';
      case 'remote':
        return 'Remote Count';
      case 'volume':
      default:
        return 'Total Jobs';
    }
  }

  String _formatSourceName(String source) {
    if (source.isEmpty) return 'Unknown';
    final normalized = source.replaceAll('_', ' ').replaceAll('-', ' ');
    final knownUppercase = {'hn hiring': 'HN Hiring', 'api': 'API', 'yc': 'YC'};
    if (knownUppercase.containsKey(normalized.toLowerCase())) {
      return knownUppercase[normalized.toLowerCase()]!;
    }
    return normalized
        .split(' ')
        .map((word) => word.isEmpty ? '' : '${word[0].toUpperCase()}${word.substring(1).toLowerCase()}')
        .join(' ');
  }
}
