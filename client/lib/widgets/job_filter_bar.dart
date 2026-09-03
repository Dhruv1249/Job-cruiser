import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../models/job_filter_state.dart';

/// Compact, horizontal quick filter chip bar with mouse wheel, drag, and arrow navigation scrolling support.
class JobFilterBar extends StatefulWidget {
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
  State<JobFilterBar> createState() => _JobFilterBarState();
}

class _JobFilterBarState extends State<JobFilterBar> {
  late final ScrollController _scrollController;
  bool _canScrollLeft = false;
  bool _canScrollRight = false;

  @override
  void initState() {
    super.initState();
    _scrollController = ScrollController();
    _scrollController.addListener(_updateScrollIndicators);
    WidgetsBinding.instance.addPostFrameCallback((_) => _updateScrollIndicators());
  }

  @override
  void dispose() {
    _scrollController.removeListener(_updateScrollIndicators);
    _scrollController.dispose();
    super.dispose();
  }

  void _updateScrollIndicators() {
    if (!_scrollController.hasClients) return;
    final position = _scrollController.position;
    final canLeft = position.pixels > 2.0;
    final canRight = position.pixels < (position.maxScrollExtent - 2.0);
    if (canLeft != _canScrollLeft || canRight != _canScrollRight) {
      setState(() {
        _canScrollLeft = canLeft;
        _canScrollRight = canRight;
      });
    }
  }

  void _scrollBy(double offset) {
    if (!_scrollController.hasClients) return;
    final target = (_scrollController.offset + offset).clamp(
      0.0,
      _scrollController.position.maxScrollExtent,
    );
    _scrollController.animateTo(
      target,
      duration: const Duration(milliseconds: 250),
      curve: Curves.easeInOut,
    );
  }

  @override
  Widget build(BuildContext context) {
    final activeCount = widget.filterState.activeFilterCount;
    final isCustomScore = widget.filterState.minScore > 0 || widget.filterState.maxScore < 100;
    final isCustomRecency = widget.filterState.recencyDays != null && widget.filterState.recencyDays! > 0;

    return Container(
      color: AppColors.surface,
      height: 44,
      child: NotificationListener<ScrollMetricsNotification>(
        onNotification: (notification) {
          _updateScrollIndicators();
          return false;
        },
        child: Listener(
          onPointerSignal: (pointerSignal) {
            if (pointerSignal is PointerScrollEvent && _scrollController.hasClients) {
              final delta = pointerSignal.scrollDelta.dy != 0
                  ? pointerSignal.scrollDelta.dy
                  : pointerSignal.scrollDelta.dx;
              final target = (_scrollController.offset + delta).clamp(
                0.0,
                _scrollController.position.maxScrollExtent,
              );
              _scrollController.jumpTo(target);
            }
          },
          child: Stack(
            children: [
              SingleChildScrollView(
                controller: _scrollController,
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
                    if (!widget.filterState.isDefault) ...[
                      const SizedBox(width: 8),
                      _buildClearAllChip(),
                    ],
                  ],
                ),
              ),
              if (_canScrollLeft)
                Positioned(
                  left: 0,
                  top: 0,
                  bottom: 0,
                  child: _buildScrollArrow(
                    icon: Icons.chevron_left,
                    onPressed: () => _scrollBy(-140),
                    alignment: Alignment.centerLeft,
                  ),
                ),
              if (_canScrollRight)
                Positioned(
                  right: 0,
                  top: 0,
                  bottom: 0,
                  child: _buildScrollArrow(
                    icon: Icons.chevron_right,
                    onPressed: () => _scrollBy(140),
                    alignment: Alignment.centerRight,
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }

  Widget _buildScrollArrow({
    required IconData icon,
    required VoidCallback onPressed,
    required Alignment alignment,
  }) {
    final isLeft = alignment == Alignment.centerLeft;
    return Container(
      width: 32,
      decoration: BoxDecoration(
        gradient: LinearGradient(
          begin: isLeft ? Alignment.centerLeft : Alignment.centerRight,
          end: isLeft ? Alignment.centerRight : Alignment.centerLeft,
          colors: [
            AppColors.surface,
            AppColors.surface.withValues(alpha: 0.9),
            AppColors.surface.withValues(alpha: 0.0),
          ],
        ),
      ),
      alignment: alignment,
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: onPressed,
          borderRadius: BorderRadius.circular(16),
          child: Container(
            width: 24,
            height: 24,
            decoration: BoxDecoration(
              color: AppColors.surfaceContainerLowest,
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.12),
                  blurRadius: 4,
                  offset: const Offset(0, 1),
                ),
              ],
            ),
            child: Icon(icon, size: 16, color: AppColors.primary),
          ),
        ),
      ),
    );
  }

  Widget _buildAllFiltersButton(int activeCount) {
    final hasActive = activeCount > 0;
    return OutlinedButton.icon(
      onPressed: widget.onOpenFilterDialog,
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
    final isDateSort = widget.filterState.sortBy == 'date_desc';
    final isSalarySort = widget.filterState.sortBy == 'salary_desc';
    final isOldestSort = widget.filterState.sortBy == 'date_asc';

    String label = 'Sort: Score ↓';
    if (isDateSort) label = 'Sort: Date ↓';
    if (isSalarySort) label = 'Sort: Salary ↓';
    if (isOldestSort) label = 'Sort: Oldest ↑';

    return PopupMenuButton<String>(
      tooltip: 'Change sorting order',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (selectedSort) {
        widget.onFilterChanged(widget.filterState.copyWith(sortBy: selectedSort));
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
    final scope = widget.filterState.matchScope;
    String label = 'All Jobs';
    if (scope == 'matched_only') label = 'Matched Only';
    if (scope == 'unmatched_only') label = 'Unmatched Only';

    return PopupMenuButton<String>(
      tooltip: 'Filter by match scope',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (selectedScope) {
        widget.onFilterChanged(widget.filterState.copyWith(matchScope: selectedScope));
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
    if (widget.filterState.minScore == 90 && widget.filterState.maxScore == 100) label = '90%+ Elite';
    if (widget.filterState.minScore == 80 && widget.filterState.maxScore == 100) label = '80%+ Top';
    if (widget.filterState.minScore == 60 && widget.filterState.maxScore == 100) label = '60%+ Good';
    if (isCustomScore && label == 'All Scores') {
      label = '${widget.filterState.minScore}%-${widget.filterState.maxScore}%';
    }

    return PopupMenuButton<String>(
      tooltip: 'Filter by score range',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (value) {
        if (value == '0') {
          widget.onFilterChanged(widget.filterState.copyWith(minScore: 0, maxScore: 100));
        } else if (value == '60') {
          widget.onFilterChanged(widget.filterState.copyWith(minScore: 60, maxScore: 100));
        } else if (value == '80') {
          widget.onFilterChanged(widget.filterState.copyWith(minScore: 80, maxScore: 100));
        } else if (value == '90') {
          widget.onFilterChanged(widget.filterState.copyWith(minScore: 90, maxScore: 100));
        } else if (value == 'custom') {
          widget.onOpenFilterDialog();
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
    final days = widget.filterState.recencyDays;
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
          widget.onFilterChanged(widget.filterState.copyWith(recencyDays: () => null));
        } else if (value == 'custom') {
          widget.onOpenFilterDialog();
        } else {
          final parsed = int.tryParse(value);
          if (parsed != null) {
            widget.onFilterChanged(widget.filterState.copyWith(recencyDays: () => parsed));
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
    final isUnviewed = widget.filterState.viewMode == 'unviewed';
    final isViewed = widget.filterState.viewMode == 'viewed';
    final isCustomView = isUnviewed || isViewed;

    String label = 'All Views';
    if (isUnviewed) label = 'Unviewed';
    if (isViewed) label = 'Viewed';

    return PopupMenuButton<String>(
      tooltip: 'Filter by view status',
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(12)),
      onSelected: (mode) {
        widget.onFilterChanged(widget.filterState.copyWith(viewMode: mode));
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
        widget.onFilterChanged(widget.filterState.reset());
      },
    );
  }
}
