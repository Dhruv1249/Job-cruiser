import 'package:flutter/material.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Renders a ranked leaderboard of job sources by AI matching quality and yield.
class SourceQualityLeaderboard extends StatefulWidget {
  const SourceQualityLeaderboard({
    super.key,
    required this.sourcesQuality,
  });

  final List<SourceQualityMetric> sourcesQuality;

  @override
  State<SourceQualityLeaderboard> createState() => _SourceQualityLeaderboardState();
}

class _SourceQualityLeaderboardState extends State<SourceQualityLeaderboard> {
  String _activeSortMode = 'score';

  @override
  Widget build(BuildContext context) {
    if (widget.sourcesQuality.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
        ),
        child: const Center(
          child: Text(
            'No AI match quality metrics recorded yet. Trigger AI evaluation passes to populate rankings.',
            textAlign: TextAlign.center,
            style: TextStyle(color: AppColors.onSurfaceVariant),
          ),
        ),
      );
    }

    final sortedList = List<SourceQualityMetric>.from(widget.sourcesQuality);
    if (_activeSortMode == 'score') {
      sortedList.sort((a, b) => b.avgScore.compareTo(a.avgScore));
    } else if (_activeSortMode == 'yield') {
      sortedList.sort((a, b) => b.highMatchYieldPct.compareTo(a.highMatchYieldPct));
    } else if (_activeSortMode == 'volume') {
      sortedList.sort((a, b) => b.evaluatedCount.compareTo(a.evaluatedCount));
    }

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
                      'AI Match Quality Rankings',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        color: AppColors.primary,
                        letterSpacing: -0.3,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Which scraping targets produce the strongest profile alignment',
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
                  PopupMenuItem(value: 'score', child: Text('Sort by Avg Score')),
                  PopupMenuItem(value: 'yield', child: Text('Sort by High-Match Yield %')),
                  PopupMenuItem(value: 'volume', child: Text('Sort by Evaluated Count')),
                ],
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
                  decoration: BoxDecoration(
                    color: AppColors.surfaceContainerHigh,
                    borderRadius: BorderRadius.circular(8),
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(Icons.sort, size: 14, color: AppColors.primary),
                      const SizedBox(width: 4),
                      Text(
                        _resolveSortLabel(_activeSortMode),
                        style: const TextStyle(
                          fontSize: 11,
                          fontWeight: FontWeight.w600,
                          color: AppColors.primary,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          ListView.separated(
            itemCount: sortedList.length,
            shrinkWrap: true,
            physics: const NeverScrollableScrollPhysics(),
            separatorBuilder: (context, index) => const Divider(
              height: 14,
              color: AppColors.surfaceContainerHigh,
            ),
            itemBuilder: (context, index) {
              final metric = sortedList[index];
              return _buildRankTile(metric, index + 1);
            },
          ),
        ],
      ),
    );
  }

  Widget _buildRankTile(SourceQualityMetric metric, int rank) {
    final rankColor = _resolveRankColor(rank);

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Container(
          width: 26,
          height: 26,
          alignment: Alignment.center,
          decoration: BoxDecoration(
            color: rankColor.withValues(alpha: 0.12),
            shape: BoxShape.circle,
          ),
          child: Text(
            '$rank',
            style: TextStyle(
              fontSize: 11,
              fontWeight: FontWeight.w800,
              color: rankColor,
            ),
          ),
        ),
        const SizedBox(width: 12),
        Expanded(
          flex: 3,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                metric.source.toUpperCase(),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  color: AppColors.primary,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                '${metric.evaluatedCount} evaluated • ${metric.eliteMatches} elite (80%+)',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 11,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
        const SizedBox(width: 10),
        Expanded(
          flex: 2,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  Container(
                    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                    decoration: BoxDecoration(
                      color: _resolveScoreColor(metric.avgScore).withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Text(
                      '${metric.avgScore.toStringAsFixed(1)}% Avg',
                      style: TextStyle(
                        fontSize: 11,
                        fontWeight: FontWeight.w800,
                        color: _resolveScoreColor(metric.avgScore),
                      ),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 4),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  SizedBox(
                    width: 60,
                    child: ClipRRect(
                      borderRadius: BorderRadius.circular(4),
                      child: LinearProgressIndicator(
                        value: (metric.highMatchYieldPct / 100).clamp(0.0, 1.0),
                        backgroundColor: AppColors.outlineVariant.withValues(alpha: 0.3),
                        valueColor: AlwaysStoppedAnimation<Color>(_resolveScoreColor(metric.avgScore)),
                        minHeight: 5,
                      ),
                    ),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    '${metric.highMatchYieldPct.toStringAsFixed(0)}% yield',
                    style: const TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w600,
                      color: AppColors.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ],
    );
  }

  Color _resolveRankColor(int rank) {
    if (rank == 1) return const Color(0xFFEAB308);
    if (rank == 2) return const Color(0xFF94A3B8);
    if (rank == 3) return const Color(0xFFD97706);
    return AppColors.outline;
  }

  Color _resolveScoreColor(double score) {
    if (score >= 80) return AppColors.matchGreen;
    if (score >= 60) return AppColors.secondary;
    return AppColors.error;
  }

  String _resolveSortLabel(String sortMode) {
    if (sortMode == 'yield') return 'Yield %';
    if (sortMode == 'volume') return 'Evaluated';
    return 'Avg Score';
  }
}
