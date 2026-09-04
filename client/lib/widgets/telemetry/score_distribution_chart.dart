import 'package:flutter/material.dart';
import 'package:fl_chart/fl_chart.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Renders an interactive score distribution donut chart and top hiring companies discovered.
class ScoreDistributionChart extends StatefulWidget {
  const ScoreDistributionChart({
    super.key,
    required this.scoreDistribution,
    required this.topCompanies,
  });

  final ScoreDistributionMetric scoreDistribution;
  final List<TopCompanyMetric> topCompanies;

  @override
  State<ScoreDistributionChart> createState() => _ScoreDistributionChartState();
}

class _ScoreDistributionChartState extends State<ScoreDistributionChart> {
  int _touchedIndex = -1;

  @override
  Widget build(BuildContext context) {
    final distribution = widget.scoreDistribution;
    final total = distribution.totalEvaluated + distribution.unevaluatedCount;

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
      child: LayoutBuilder(
        builder: (context, constraints) {
          final isWide = constraints.maxWidth >= 960;

          return Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'AI Match Score Distribution & Top Companies',
                style: TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w700,
                  color: AppColors.primary,
                  letterSpacing: -0.3,
                ),
              ),
              const SizedBox(height: 2),
              Text(
                'Overall distribution of job match tiers across the ingestion database',
                style: TextStyle(
                  fontSize: 12,
                  color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
                ),
              ),
              const SizedBox(height: 20),
              if (isWide)
                Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      flex: 5,
                      child: _buildChartSection(distribution, total),
                    ),
                    const SizedBox(width: 24),
                    Container(width: 1, height: 180, color: AppColors.surfaceContainer),
                    const SizedBox(width: 24),
                    Expanded(
                      flex: 4,
                      child: _buildTopCompaniesSection(widget.topCompanies),
                    ),
                  ],
                )
              else
                Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildChartSection(distribution, total),
                    const SizedBox(height: 24),
                    const Divider(height: 1, color: AppColors.surfaceContainer),
                    const SizedBox(height: 16),
                    _buildTopCompaniesSection(widget.topCompanies),
                  ],
                ),
            ],
          );
        },
      ),
    );
  }

  Widget _buildChartSection(ScoreDistributionMetric distribution, int total) {
    if (total == 0) {
      return const SizedBox(
        height: 180,
        child: Center(
          child: Text(
            'No score distribution records available.',
            style: TextStyle(color: AppColors.onSurfaceVariant),
          ),
        ),
      );
    }

    final tiers = [
      _TierItem(
        label: '90–100% Elite',
        count: distribution.tier90To100,
        color: const Color(0xFF10B981),
      ),
      _TierItem(
        label: '80–89% Top Match',
        count: distribution.tier80To89,
        color: const Color(0xFF14B8A6),
      ),
      _TierItem(
        label: '60–79% Good Match',
        count: distribution.tier60To79,
        color: const Color(0xFFF59E0B),
      ),
      _TierItem(
        label: '<60% Low Match',
        count: distribution.tierBelow60,
        color: const Color(0xFF64748B),
      ),
      _TierItem(
        label: 'Unevaluated',
        count: distribution.unevaluatedCount,
        color: const Color(0xFFCBD5E1),
      ),
    ];

    return Row(
      children: [
        SizedBox(
          width: 160,
          height: 160,
          child: PieChart(
            PieChartData(
              pieTouchData: PieTouchData(
                touchCallback: (event, pieTouchResponse) {
                  setState(() {
                    if (!event.isInterestedForInteractions ||
                        pieTouchResponse == null ||
                        pieTouchResponse.touchedSection == null) {
                      _touchedIndex = -1;
                      return;
                    }
                    _touchedIndex = pieTouchResponse.touchedSection!.touchedSectionIndex;
                  });
                },
              ),
              borderData: FlBorderData(show: false),
              sectionsSpace: 2,
              centerSpaceRadius: 42,
              sections: List.generate(tiers.length, (index) {
                final tier = tiers[index];
                final isTouched = index == _touchedIndex;
                final radius = isTouched ? 34.0 : 28.0;
                final double value = tier.count.toDouble();

                return PieChartSectionData(
                  color: tier.color,
                  value: value > 0 ? value : 0.0001,
                  title: '',
                  radius: radius,
                );
              }),
            ),
          ),
        ),
        const SizedBox(width: 20),
        Expanded(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: tiers.asMap().entries.map((entry) {
              final index = entry.key;
              final tier = entry.value;
              final pct = total > 0 ? (tier.count * 100.0 / total) : 0.0;
              final isHighlighted = index == _touchedIndex;

              return Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Row(
                  children: [
                    Container(
                      width: 10,
                      height: 10,
                      decoration: BoxDecoration(
                        color: tier.color,
                        shape: BoxShape.circle,
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        tier.label,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 12,
                          fontWeight: isHighlighted ? FontWeight.w700 : FontWeight.w500,
                          color: AppColors.primary,
                        ),
                      ),
                    ),
                    Text(
                      '${tier.count}',
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: isHighlighted ? AppColors.primary : AppColors.secondary,
                      ),
                    ),
                    const SizedBox(width: 6),
                    Text(
                      '(${pct.toStringAsFixed(1)}%)',
                      style: TextStyle(
                        fontSize: 11,
                        color: AppColors.onSurfaceVariant.withValues(alpha: 0.7),
                      ),
                    ),
                  ],
                ),
              );
            }).toList(),
          ),
        ),
      ],
    );
  }

  Widget _buildTopCompaniesSection(List<TopCompanyMetric> topCompanies) {
    if (topCompanies.isEmpty) {
      return const SizedBox(
        height: 100,
        child: Center(
          child: Text(
            'No company statistics available.',
            style: TextStyle(color: AppColors.onSurfaceVariant),
          ),
        ),
      );
    }

    final displayedCompanies = topCompanies.take(6).toList();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Expanded(
              child: Text(
                'Top Discovered Hiring Cos',
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w700,
                  color: AppColors.primary,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Text(
              '${topCompanies.length} total',
              style: TextStyle(
                fontSize: 11,
                color: AppColors.onSurfaceVariant.withValues(alpha: 0.7),
              ),
            ),
          ],
        ),
        const SizedBox(height: 10),
        ListView.separated(
          shrinkWrap: true,
          physics: const NeverScrollableScrollPhysics(),
          itemCount: displayedCompanies.length,
          separatorBuilder: (context, index) => const Divider(height: 10, color: AppColors.surfaceContainer),
          itemBuilder: (context, index) {
            final company = displayedCompanies[index];
            return Row(
              children: [
                Container(
                  width: 20,
                  height: 20,
                  alignment: Alignment.center,
                  decoration: BoxDecoration(
                    color: AppColors.surfaceContainerLow,
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    '${index + 1}',
                    style: const TextStyle(
                      fontSize: 10,
                      fontWeight: FontWeight.w700,
                      color: AppColors.secondary,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    company.companyName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 12,
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
                    '${company.jobCount} roles',
                    style: const TextStyle(
                      fontSize: 11,
                      fontWeight: FontWeight.w600,
                      color: AppColors.primary,
                    ),
                  ),
                ),
              ],
            );
          },
        ),
      ],
    );
  }
}

class _TierItem {
  const _TierItem({
    required this.label,
    required this.count,
    required this.color,
  });

  final String label;
  final int count;
  final Color color;
}
