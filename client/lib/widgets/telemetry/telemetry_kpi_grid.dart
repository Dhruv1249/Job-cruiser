import 'package:flutter/material.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Renders a responsive grid of key performance indicators for scraper operations.
class TelemetryKpiGrid extends StatelessWidget {
  const TelemetryKpiGrid({
    super.key,
    required this.kpis,
  });

  final TelemetryKpis kpis;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final screenWidth = constraints.maxWidth;
        final isCompact = screenWidth < 600;
        final isMedium = screenWidth >= 600 && screenWidth < 960;

        final crossAxisCount = isCompact ? 2 : (isMedium ? 3 : 6);
        final cardAspectRatio = isCompact ? 1.35 : (isMedium ? 1.5 : 1.25);

        return GridView.count(
          crossAxisCount: crossAxisCount,
          crossAxisSpacing: 10,
          mainAxisSpacing: 10,
          childAspectRatio: cardAspectRatio,
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          children: [
            _buildMetricCard(
              title: 'Total Ingested',
              value: '${kpis.totalJobs}',
              badgeText: '+${kpis.jobsLast24h} 24h',
              badgeColor: AppColors.matchGreen,
              icon: Icons.work_outline,
              accentColor: AppColors.primary,
            ),
            _buildMetricCard(
              title: 'Unique Companies',
              value: '${kpis.uniqueCompanies}',
              badgeText: '${kpis.jobsLast7d} 7d',
              badgeColor: AppColors.secondary,
              icon: Icons.business_outlined,
              accentColor: AppColors.secondary,
            ),
            _buildMetricCard(
              title: 'AI Evaluated',
              value: '${kpis.evaluationCoveragePct.toStringAsFixed(1)}%',
              badgeText: '${kpis.evaluatedJobsCount} jobs',
              badgeColor: AppColors.primaryContainer,
              icon: Icons.auto_awesome_outlined,
              accentColor: AppColors.primary,
            ),
            _buildMetricCard(
              title: 'Avg Match Score',
              value: '${kpis.overallAvgMatchScore.toStringAsFixed(1)}%',
              badgeText: _resolveGradeBadge(kpis.overallAvgMatchScore),
              badgeColor: _resolveScoreColor(kpis.overallAvgMatchScore),
              icon: Icons.speed_outlined,
              accentColor: _resolveScoreColor(kpis.overallAvgMatchScore),
            ),
            _buildMetricCard(
              title: 'Top Volume Source',
              value: kpis.topVolumeSource.toUpperCase(),
              badgeText: 'Leader',
              badgeColor: AppColors.primary,
              icon: Icons.bar_chart_outlined,
              accentColor: AppColors.primary,
            ),
            _buildMetricCard(
              title: 'Best Quality Source',
              value: kpis.topQualitySource.toUpperCase(),
              badgeText: 'Highest Yield',
              badgeColor: AppColors.matchGreen,
              icon: Icons.workspace_premium_outlined,
              accentColor: AppColors.matchGreen,
            ),
          ],
        );
      },
    );
  }

  Widget _buildMetricCard({
    required String title,
    required String value,
    required String badgeText,
    required Color badgeColor,
    required IconData icon,
    required Color accentColor,
  }) {
    return Container(
      padding: const EdgeInsets.all(12),
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
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Container(
                padding: const EdgeInsets.all(6),
                decoration: BoxDecoration(
                  color: accentColor.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Icon(icon, size: 16, color: accentColor),
              ),
              Flexible(
                child: Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: badgeColor.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: Text(
                    badgeText,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                      color: badgeColor,
                    ),
                  ),
                ),
              ),
            ],
          ),
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                value,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w800,
                  color: AppColors.primary,
                  letterSpacing: -0.5,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                title,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Color _resolveScoreColor(double score) {
    if (score >= 80) return AppColors.matchGreen;
    if (score >= 60) return AppColors.secondary;
    return AppColors.error;
  }

  String _resolveGradeBadge(double score) {
    if (score >= 80) return 'Top Tier';
    if (score >= 60) return 'Good Match';
    return 'Calibrating';
  }
}
