import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../models/job_filter_state.dart';

/// Compact, horizontal quick filter chip bar for real-time filter tweaking and sorting toggling.
class JobFilterBar extends StatelessWidget {
  const JobFilterBar({
    super.key,
    required this.filterState,
    required this.onFilterChanged,
    required this.onOpenFilterDialog,
  });

  final JobFilterState filterState;
  final ValueChanged<JobFilterState> onFilterChanged;
  final VoidCallback onOpenFilterDialog;

  @override
  Widget build(BuildContext context) {
    final activeCount = filterState.activeFilterCount;
    final isCustomScore = filterState.minScore > 0 || filterState.maxScore < 100;
    final isCustomRecency = filterState.recencyDays != null && filterState.recencyDays! > 0;

    return Container(
      color: AppColors.surface,
      height: 44,
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 16),
        child: Row(
          children: [
            _buildAllFiltersButton(activeCount),
            const SizedBox(width: 8),
            _buildQuickSortChip(),
            const SizedBox(width: 6),
            _buildScopeSelectorChip(),
            const SizedBox(width: 6),
            _buildScorePresetChip(isCustomScore),
            const SizedBox(width: 6),
            _buildRecencyPresetChip(isCustomRecency),
            const SizedBox(width: 6),
            _buildViewStatusChip(),
            if (!filterState.isDefault) ...[
              const SizedBox(width: 8),
              _buildClearAllChip(),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildAllFiltersButton(int activeCount) {
    final hasActive = activeCount > 0;
    return OutlinedButton.icon(
      onPressed: onOpenFilterDialog,
      icon: Icon(
        Icons.tune,
        size: 14,
        color: hasActive ? Colors.white : AppColors.primary,
      ),
      label: Text(
        hasActive ? 'Filters ($activeCount)' : 'All Filters',
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w700,
          color: hasActive ? Colors.white : AppColors.primary,
        ),
      ),
      style: OutlinedButton.styleFrom(
        backgroundColor: hasActive ? AppColors.primary : AppColors.surfaceContainerLowest,
        side: BorderSide(
          color: hasActive ? AppColors.primary : AppColors.outlineVariant,
        ),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 0),
        minimumSize: const Size(0, 32),
        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      ),
    );
  }

  Widget _buildQuickSortChip() {
    final isDateSort = filterState.sortBy == 'date_desc';
    final isSalarySort = filterState.sortBy == 'salary_desc';
    final isOldestSort = filterState.sortBy == 'date_asc';

    String label = 'Sort: Score ↓';
    if (isDateSort) label = 'Sort: Date ↓';
    if (isSalarySort) label = 'Sort: Salary ↓';
    if (isOldestSort) label = 'Sort: Oldest ↑';

    return PopupMenuButton<String>(
      tooltip: 'Change sorting order',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (selectedSort) {
        onFilterChanged(filterState.copyWith(sortBy: selectedSort));
      },
      itemBuilder: (context) => const [
        PopupMenuItem(
          value: 'score_desc',
          child: Row(
            children: [
              Icon(Icons.bolt, size: 16, color: AppColors.primary),
              SizedBox(width: 8),
              Text('Match Score (Highest First)'),
            ],
          ),
        ),
        PopupMenuItem(
          value: 'date_desc',
          child: Row(
            children: [
              Icon(Icons.schedule, size: 16, color: AppColors.primary),
              SizedBox(width: 8),
              Text('Date (Newest First)'),
            ],
          ),
        ),
        PopupMenuItem(
          value: 'date_asc',
          child: Row(
            children: [
              Icon(Icons.history, size: 16, color: AppColors.primary),
              SizedBox(width: 8),
              Text('Date (Oldest First)'),
            ],
          ),
        ),
        PopupMenuItem(
          value: 'salary_desc',
          child: Row(
            children: [
              Icon(Icons.attach_money, size: 16, color: AppColors.primary),
              SizedBox(width: 8),
              Text('Salary (Highest First)'),
            ],
          ),
        ),
      ],
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(color: AppColors.outlineVariant),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              isDateSort
                  ? Icons.schedule
                  : isSalarySort
                      ? Icons.attach_money
                      : Icons.bolt,
              size: 13,
              color: AppColors.primary,
            ),
            const SizedBox(width: 4),
            Text(
              label,
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w700,
                color: AppColors.primary,
              ),
            ),
            const SizedBox(width: 2),
            const Icon(Icons.arrow_drop_down, size: 16, color: AppColors.onSurfaceVariant),
          ],
        ),
      ),
    );
  }

  Widget _buildScopeSelectorChip() {
    final scope = filterState.matchScope;
    String label = 'All Jobs';
    if (scope == 'matched_only') label = 'Matched Only';
    if (scope == 'unmatched_only') label = 'Unmatched Only';

    return PopupMenuButton<String>(
      tooltip: 'Filter by match scope',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (selectedScope) {
        onFilterChanged(filterState.copyWith(matchScope: selectedScope));
      },
      itemBuilder: (context) => const [
        PopupMenuItem(value: 'all', child: Text('All Jobs (Both)')),
        PopupMenuItem(value: 'matched_only', child: Text('Matched Only')),
        PopupMenuItem(value: 'unmatched_only', child: Text('Unmatched Only')),
      ],
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: scope != 'all' ? AppColors.primary : AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: scope != 'all' ? AppColors.primary : AppColors.outlineVariant,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: scope != 'all' ? Colors.white : AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 2),
            Icon(
              Icons.arrow_drop_down,
              size: 16,
              color: scope != 'all' ? Colors.white : AppColors.onSurfaceVariant,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildScorePresetChip(bool isCustomScore) {
    String label = 'All Scores';
    if (filterState.minScore == 90 && filterState.maxScore == 100) label = '90%+ Elite';
    if (filterState.minScore == 80 && filterState.maxScore == 100) label = '80%+ Top';
    if (filterState.minScore == 60 && filterState.maxScore == 100) label = '60%+ Good';
    if (isCustomScore && label == 'All Scores') {
      label = '${filterState.minScore}%-${filterState.maxScore}%';
    }

    return PopupMenuButton<String>(
      tooltip: 'Filter by score range',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (value) {
        if (value == '0') {
          onFilterChanged(filterState.copyWith(minScore: 0, maxScore: 100));
        } else if (value == '60') {
          onFilterChanged(filterState.copyWith(minScore: 60, maxScore: 100));
        } else if (value == '80') {
          onFilterChanged(filterState.copyWith(minScore: 80, maxScore: 100));
        } else if (value == '90') {
          onFilterChanged(filterState.copyWith(minScore: 90, maxScore: 100));
        } else if (value == 'custom') {
          onOpenFilterDialog();
        }
      },
      itemBuilder: (context) => const [
        PopupMenuItem(value: '0', child: Text('All Scores (0%+)')),
        PopupMenuItem(value: '60', child: Text('60%+ Good Match')),
        PopupMenuItem(value: '80', child: Text('80%+ Top Match')),
        PopupMenuItem(value: '90', child: Text('90%+ Elite Match')),
        PopupMenuDivider(),
        PopupMenuItem(value: 'custom', child: Text('Custom Score Range...')),
      ],
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: isCustomScore ? AppColors.primary : AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isCustomScore ? AppColors.primary : AppColors.outlineVariant,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: isCustomScore ? Colors.white : AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 2),
            Icon(
              Icons.arrow_drop_down,
              size: 16,
              color: isCustomScore ? Colors.white : AppColors.onSurfaceVariant,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildRecencyPresetChip(bool isCustomRecency) {
    String label = 'Any Time';
    final days = filterState.recencyDays;
    if (days == 1) label = 'Today (24h)';
    if (days == 2) label = '2 Days Ago';
    if (days == 3) label = '3 Days Ago';
    if (days == 7) label = 'Past Week';
    if (days == 14) label = 'Past 2 Weeks';
    if (days != null && days > 0 && label == 'Any Time') label = '$days Days Ago';

    return PopupMenuButton<String>(
      tooltip: 'Filter by posting date',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (value) {
        if (value == 'null') {
          onFilterChanged(filterState.copyWith(recencyDays: () => null));
        } else if (value == 'custom') {
          onOpenFilterDialog();
        } else {
          final parsed = int.tryParse(value);
          if (parsed != null) {
            onFilterChanged(filterState.copyWith(recencyDays: () => parsed));
          }
        }
      },
      itemBuilder: (context) => const [
        PopupMenuItem(value: 'null', child: Text('Any Time')),
        PopupMenuItem(value: '1', child: Text('Today / Past 24h')),
        PopupMenuItem(value: '2', child: Text('Past 2 Days')),
        PopupMenuItem(value: '3', child: Text('Past 3 Days')),
        PopupMenuItem(value: '7', child: Text('Past Week (7d)')),
        PopupMenuItem(value: '14', child: Text('Past 2 Weeks (14d)')),
        PopupMenuDivider(),
        PopupMenuItem(value: 'custom', child: Text('Custom Days...')),
      ],
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: isCustomRecency ? AppColors.primary : AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isCustomRecency ? AppColors.primary : AppColors.outlineVariant,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: isCustomRecency ? Colors.white : AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 2),
            Icon(
              Icons.arrow_drop_down,
              size: 16,
              color: isCustomRecency ? Colors.white : AppColors.onSurfaceVariant,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildViewStatusChip() {
    final isUnviewed = filterState.viewMode == 'unviewed';
    final isViewed = filterState.viewMode == 'viewed';
    final isCustomView = isUnviewed || isViewed;

    String label = 'All Views';
    if (isUnviewed) label = 'Unviewed';
    if (isViewed) label = 'Viewed';

    return PopupMenuButton<String>(
      tooltip: 'Filter by view status',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (mode) {
        onFilterChanged(filterState.copyWith(viewMode: mode));
      },
      itemBuilder: (context) => const [
        PopupMenuItem(value: 'all', child: Text('All (Viewed & Unviewed)')),
        PopupMenuItem(value: 'unviewed', child: Text('Unviewed Only')),
        PopupMenuItem(value: 'viewed', child: Text('Viewed Only')),
      ],
      child: Container(
        height: 32,
        padding: const EdgeInsets.symmetric(horizontal: 10),
        decoration: BoxDecoration(
          color: isCustomView ? AppColors.primary : AppColors.surfaceContainerLowest,
          borderRadius: BorderRadius.circular(16),
          border: Border.all(
            color: isCustomView ? AppColors.primary : AppColors.outlineVariant,
          ),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: isCustomView ? Colors.white : AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(width: 2),
            Icon(
              Icons.arrow_drop_down,
              size: 16,
              color: isCustomView ? Colors.white : AppColors.onSurfaceVariant,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildClearAllChip() {
    return ActionChip(
      avatar: const Icon(Icons.close, size: 14, color: AppColors.error),
      label: const Text('Reset'),
      labelStyle: const TextStyle(
        fontSize: 11,
        fontWeight: FontWeight.w700,
        color: AppColors.error,
      ),
      backgroundColor: AppColors.surfaceContainerLowest,
      side: const BorderSide(color: AppColors.error),
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
      onPressed: () {
        onFilterChanged(filterState.reset());
      },
    );
  }
}
