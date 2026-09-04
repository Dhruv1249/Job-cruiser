import 'dart:math' as math;
import 'package:fl_chart/fl_chart.dart';
import 'package:flutter/material.dart';
import '../../main.dart' show AppColors;
import '../../models/scraper_telemetry_models.dart';

/// Renders an interactive 14-day ingestion velocity line chart with gradient fill.
class IngestionTimelineChart extends StatelessWidget {
  const IngestionTimelineChart({
    super.key,
    required this.timeline,
  });

  final List<DailyIngestionMetric> timeline;

  @override
  Widget build(BuildContext context) {
    if (timeline.isEmpty) {
      return Container(
        padding: const EdgeInsets.all(24),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(10),
          border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
        ),
        child: const Center(
          child: Text(
            'No ingestion timeline data recorded yet.',
            style: TextStyle(color: AppColors.onSurfaceVariant),
          ),
        ),
      );
    }

    final totalIngestedInPeriod = timeline.fold<int>(0, (sum, item) => sum + item.jobsCount);
    final maxDailyCount = timeline.map((item) => item.jobsCount).fold<int>(0, math.max);

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
                      '14-Day Ingestion Velocity',
                      style: TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                        color: AppColors.primary,
                        letterSpacing: -0.3,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      'Daily new unique job postings ingested across all targets',
                      style: TextStyle(
                        fontSize: 12,
                        color: AppColors.onSurfaceVariant.withValues(alpha: 0.8),
                      ),
                    ),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                decoration: BoxDecoration(
                  color: AppColors.primaryContainer.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(20),
                ),
                child: Text(
                  '$totalIngestedInPeriod jobs total',
                  style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w700,
                    color: AppColors.primary,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 24),
          SizedBox(
            height: 200,
            child: LineChart(
              _buildChartData(maxDailyCount),
            ),
          ),
        ],
      ),
    );
  }

  LineChartData _buildChartData(int maxDailyCount) {
    final spots = <FlSpot>[];
    for (var index = 0; index < timeline.length; index++) {
      spots.add(FlSpot(index.toDouble(), timeline[index].jobsCount.toDouble()));
    }

    final double calculatedMaxY = maxDailyCount > 0 ? (maxDailyCount * 1.25).toDouble() : 10.0;

    return LineChartData(
      minX: 0,
      maxX: (timeline.length - 1).toDouble(),
      minY: 0,
      maxY: calculatedMaxY,
      gridData: FlGridData(
        show: true,
        drawVerticalLine: false,
        horizontalInterval: calculatedMaxY > 4 ? calculatedMaxY / 4 : 1,
        getDrawingHorizontalLine: (value) => FlLine(
          color: AppColors.outlineVariant.withValues(alpha: 0.3),
          strokeWidth: 1,
        ),
      ),
      titlesData: FlTitlesData(
        topTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
        rightTitles: const AxisTitles(sideTitles: SideTitles(showTitles: false)),
        leftTitles: AxisTitles(
          sideTitles: SideTitles(
            showTitles: true,
            reservedSize: 36,
            interval: calculatedMaxY > 4 ? calculatedMaxY / 4 : 1,
            getTitlesWidget: (value, meta) {
              if (value == 0) return const SizedBox.shrink();
              return Text(
                value.toInt().toString(),
                style: const TextStyle(
                  fontSize: 10,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurfaceVariant,
                ),
              );
            },
          ),
        ),
        bottomTitles: AxisTitles(
          sideTitles: SideTitles(
            showTitles: true,
            reservedSize: 24,
            interval: math.max(1, (timeline.length / 7).floor()).toDouble(),
            getTitlesWidget: (value, meta) {
              final index = value.toInt();
              if (index < 0 || index >= timeline.length) {
                return const SizedBox.shrink();
              }
              final rawDate = timeline[index].date;
              final shortDate = _formatShortDate(rawDate);
              return Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(
                  shortDate,
                  style: const TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.w600,
                    color: AppColors.onSurfaceVariant,
                  ),
                ),
              );
            },
          ),
        ),
      ),
      borderData: FlBorderData(show: false),
      lineTouchData: LineTouchData(
        handleBuiltInTouches: true,
        touchTooltipData: LineTouchTooltipData(
          getTooltipItems: (touchedSpots) {
            return touchedSpots.map((spot) {
              final index = spot.x.toInt();
              final dateLabel = index >= 0 && index < timeline.length
                  ? timeline[index].date
                  : '';
              return LineTooltipItem(
                '${spot.y.toInt()} jobs\n$dateLabel',
                const TextStyle(
                  color: Colors.white,
                  fontWeight: FontWeight.bold,
                  fontSize: 11,
                ),
              );
            }).toList();
          },
        ),
      ),
      lineBarsData: [
        LineChartBarData(
          spots: spots,
          isCurved: true,
          curveSmoothness: 0.35,
          color: AppColors.primary,
          barWidth: 3,
          isStrokeCapRound: true,
          dotData: FlDotData(
            show: timeline.length <= 14,
            getDotPainter: (spot, percent, barData, index) {
              return FlDotCirclePainter(
                radius: 3.5,
                color: AppColors.surfaceContainerLowest,
                strokeWidth: 2,
                strokeColor: AppColors.primary,
              );
            },
          ),
          belowBarData: BarAreaData(
            show: true,
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: [
                AppColors.primary.withValues(alpha: 0.25),
                AppColors.primary.withValues(alpha: 0.0),
              ],
            ),
          ),
        ),
      ],
    );
  }

  String _formatShortDate(String rawDate) {
    if (rawDate.length >= 10) {
      final parts = rawDate.split('-');
      if (parts.length == 3) {
        return '${parts[1]}/${parts[2]}';
      }
    }
    return rawDate;
  }
}
