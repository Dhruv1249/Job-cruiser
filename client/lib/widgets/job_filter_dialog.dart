import 'package:flutter/material.dart';
import '../main.dart' show AppColors;
import '../models/job_filter_state.dart';

/// Interactive modal dialog and bottom sheet allowing deep customization of all job filter and sorting dimensions.
class JobFilterDialog extends StatefulWidget {
  const JobFilterDialog({
    super.key,
    required this.initialFilterState,
    required this.onApply,
  });

  final JobFilterState initialFilterState;
  final ValueChanged<JobFilterState> onApply;

  /// Shows the filter interface adaptively as a modal dialog on wide screens and a bottom sheet on mobile.
  static Future<void> show(
    BuildContext context, {
    required JobFilterState currentState,
    required ValueChanged<JobFilterState> onApply,
  }) async {
    final screenWidth = MediaQuery.of(context).size.width;
    if (screenWidth >= 768) {
      await showDialog<void>(
        context: context,
        builder: (dialogContext) => Dialog(
          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(16)),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 580, maxHeight: 720),
            child: JobFilterDialog(
              initialFilterState: currentState,
              onApply: (updated) {
                onApply(updated);
                Navigator.of(dialogContext).pop();
              },
            ),
          ),
        ),
      );
    } else {
      await showModalBottomSheet<void>(
        context: context,
        isScrollControlled: true,
        backgroundColor: Colors.transparent,
        builder: (sheetContext) => Container(
          decoration: const BoxDecoration(
            color: AppColors.surfaceContainerLowest,
            borderRadius: BorderRadius.vertical(top: Radius.circular(20)),
          ),
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(sheetContext).size.height * 0.88,
          ),
          child: JobFilterDialog(
            initialFilterState: currentState,
            onApply: (updated) {
              onApply(updated);
              Navigator.of(sheetContext).pop();
            },
          ),
        ),
      );
    }
  }

  @override
  State<JobFilterDialog> createState() => _JobFilterDialogState();
}

class _JobFilterDialogState extends State<JobFilterDialog> {
  late String _matchScope;
  late double _minScore;
  late double _maxScore;
  late int? _recencyDays;
  late String _viewMode;
  late String _workModel;
  late String _applicationStatus;
  late String _sortBy;
  late TextEditingController _customDaysController;

  final List<int?> _recencyPresets = [null, 1, 2, 3, 7, 14];

  @override
  void initState() {
    super.initState();
    final initial = widget.initialFilterState;
    _matchScope = initial.matchScope;
    _minScore = initial.minScore.toDouble().clamp(0.0, 100.0);
    _maxScore = initial.maxScore.toDouble().clamp(0.0, 100.0);
    _recencyDays = initial.recencyDays;
    _viewMode = initial.viewMode;
    _workModel = initial.workModel;
    _applicationStatus = initial.applicationStatus;
    _sortBy = initial.sortBy;

    final isPreset = _recencyPresets.contains(_recencyDays);
    final initialText = (!isPreset && _recencyDays != null && _recencyDays! > 0)
        ? _recencyDays.toString()
        : '';
    _customDaysController = TextEditingController(text: initialText);
  }

  @override
  void dispose() {
    _customDaysController.dispose();
    super.dispose();
  }

  void _handleReset() {
    setState(() {
      _matchScope = 'all';
      _minScore = 0.0;
      _maxScore = 100.0;
      _recencyDays = null;
      _viewMode = 'all';
      _workModel = 'all';
      _applicationStatus = 'all';
      _sortBy = 'score_desc';
      _customDaysController.clear();
    });
  }

  void _handleApply() {
    int? finalDays = _recencyDays;
    final customText = _customDaysController.text.trim();
    if (customText.isNotEmpty) {
      final parsed = int.tryParse(customText);
      if (parsed != null && parsed > 0) {
        finalDays = parsed;
      }
    }

    final updated = widget.initialFilterState.copyWith(
      matchScope: _matchScope,
      minScore: _minScore.toInt(),
      maxScore: _maxScore.toInt(),
      recencyDays: () => finalDays,
      viewMode: _viewMode,
      workModel: _workModel,
      applicationStatus: _applicationStatus,
      sortBy: _sortBy,
    );

    widget.onApply(updated);
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        _buildHeader(),
        const Divider(height: 1, color: AppColors.outlineVariant),
        Flexible(
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 16),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _buildSectionTitle('Sort Order'),
                const SizedBox(height: 8),
                _buildSortSelector(),
                const SizedBox(height: 20),
                _buildSectionTitle('Match Scope'),
                const SizedBox(height: 8),
                _buildMatchScopeSelector(),
                const SizedBox(height: 20),
                _buildSectionTitle('Match Score Range (${_minScore.toInt()}% – ${_maxScore.toInt()}%)'),
                const SizedBox(height: 8),
                _buildScoreRangeSlider(),
                const SizedBox(height: 20),
                _buildSectionTitle('Posting / Scraped Recency'),
                const SizedBox(height: 8),
                _buildRecencySelector(),
                const SizedBox(height: 20),
                _buildSectionTitle('Work Model'),
                const SizedBox(height: 8),
                _buildWorkModelSelector(),
                const SizedBox(height: 20),
                _buildSectionTitle('Viewing Status'),
                const SizedBox(height: 8),
                _buildViewStatusSelector(),
                const SizedBox(height: 20),
                _buildSectionTitle('Pipeline CRM Status'),
                const SizedBox(height: 8),
                _buildApplicationStatusSelector(),
                const SizedBox(height: 12),
              ],
            ),
          ),
        ),
        const Divider(height: 1, color: AppColors.outlineVariant),
        _buildFooterActions(),
      ],
    );
  }

  Widget _buildHeader() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 14),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          const Row(
            children: [
              Icon(Icons.tune, size: 20, color: AppColors.primary),
              SizedBox(width: 8),
              Text(
                'Filter & Sort Options',
                style: TextStyle(
                  fontSize: 18,
                  fontWeight: FontWeight.w700,
                  color: AppColors.primary,
                ),
              ),
            ],
          ),
          IconButton(
            icon: const Icon(Icons.close, size: 20, color: AppColors.onSurfaceVariant),
            onPressed: () => Navigator.maybePop(context),
          ),
        ],
      ),
    );
  }

  Widget _buildSectionTitle(String title) {
    return Text(
      title,
      style: const TextStyle(
        fontSize: 13,
        fontWeight: FontWeight.w700,
        color: AppColors.primary,
        letterSpacing: -0.1,
      ),
    );
  }

  Widget _buildSortSelector() {
    final sortOptions = [
      {'key': 'score_desc', 'label': 'Match Score (Highest First)', 'icon': Icons.bolt},
      {'key': 'date_desc', 'label': 'Date (Newest Scraped First)', 'icon': Icons.schedule},
      {'key': 'date_asc', 'label': 'Date (Oldest First)', 'icon': Icons.history},
      {'key': 'salary_desc', 'label': 'Salary (Highest First)', 'icon': Icons.attach_money},
    ];

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: sortOptions.map((opt) {
        final isSelected = _sortBy == opt['key'];
        return ChoiceChip(
          avatar: Icon(
            opt['icon'] as IconData,
            size: 14,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          label: Text(opt['label'] as String),
          selected: isSelected,
          selectedColor: AppColors.primary,
          labelStyle: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          onSelected: (selected) {
            if (selected) {
              setState(() => _sortBy = opt['key'] as String);
            }
          },
        );
      }).toList(),
    );
  }

  Widget _buildMatchScopeSelector() {
    final scopes = [
      {'key': 'all', 'label': 'All Jobs (Both)'},
      {'key': 'matched_only', 'label': 'Matched Only'},
      {'key': 'unmatched_only', 'label': 'Unmatched Only'},
    ];

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: scopes.map((s) {
        final isSelected = _matchScope == s['key'];
        return ChoiceChip(
          label: Text(s['label']!),
          selected: isSelected,
          selectedColor: AppColors.primary,
          labelStyle: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          onSelected: (selected) {
            if (selected) {
              setState(() => _matchScope = s['key']!);
            }
          },
        );
      }).toList(),
    );
  }

  Widget _buildScoreRangeSlider() {
    return Column(
      children: [
        RangeSlider(
          values: RangeValues(_minScore, _maxScore),
          min: 0,
          max: 100,
          divisions: 20,
          activeColor: AppColors.primary,
          inactiveColor: AppColors.surfaceContainerHigh,
          labels: RangeLabels('${_minScore.toInt()}%', '${_maxScore.toInt()}%'),
          onChanged: (values) {
            setState(() {
              _minScore = values.start;
              _maxScore = values.end;
            });
          },
        ),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              'Min: ${_minScore.toInt()}%',
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: AppColors.secondary),
            ),
            Wrap(
              spacing: 6,
              children: [
                _buildQuickScoreButton('All (0%+)', 0, 100),
                _buildQuickScoreButton('60%+', 60, 100),
                _buildQuickScoreButton('80%+', 80, 100),
                _buildQuickScoreButton('90%+', 90, 100),
              ],
            ),
            Text(
              'Max: ${_maxScore.toInt()}%',
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w600, color: AppColors.secondary),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildQuickScoreButton(String label, double min, double max) {
    final isSelected = _minScore == min && _maxScore == max;
    return InkWell(
      onTap: () => setState(() {
        _minScore = min;
        _maxScore = max;
      }),
      borderRadius: BorderRadius.circular(6),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
        decoration: BoxDecoration(
          color: isSelected ? AppColors.primary : AppColors.surfaceContainer,
          borderRadius: BorderRadius.circular(6),
        ),
        child: Text(
          label,
          style: TextStyle(
            fontSize: 11,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
        ),
      ),
    );
  }

  Widget _buildRecencySelector() {
    final presets = [
      {'days': null, 'label': 'Any Time'},
      {'days': 1, 'label': 'Today (24h)'},
      {'days': 2, 'label': '2 Days Ago'},
      {'days': 3, 'label': '3 Days Ago'},
      {'days': 7, 'label': 'Past Week (7d)'},
      {'days': 14, 'label': 'Past 2 Weeks (14d)'},
    ];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Wrap(
          spacing: 8,
          runSpacing: 8,
          children: presets.map((p) {
            final daysVal = p['days'] as int?;
            final isSelected = _recencyDays == daysVal && _customDaysController.text.isEmpty;
            return ChoiceChip(
              label: Text(p['label'] as String),
              selected: isSelected,
              selectedColor: AppColors.primary,
              labelStyle: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w600,
                color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
              ),
              onSelected: (selected) {
                if (selected) {
                  setState(() {
                    _recencyDays = daysVal;
                    _customDaysController.clear();
                  });
                }
              },
            );
          }).toList(),
        ),
        const SizedBox(height: 10),
        Row(
          children: [
            const Text(
              'Or custom:',
              style: TextStyle(fontSize: 12, color: AppColors.onSurfaceVariant),
            ),
            const SizedBox(width: 8),
            SizedBox(
              width: 90,
              height: 36,
              child: TextField(
                controller: _customDaysController,
                keyboardType: TextInputType.number,
                decoration: const InputDecoration(
                  hintText: 'e.g. 5',
                  contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 6),
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                style: const TextStyle(fontSize: 13),
                onChanged: (val) {
                  final parsed = int.tryParse(val.trim());
                  if (parsed != null && parsed > 0) {
                    setState(() => _recencyDays = parsed);
                  }
                },
              ),
            ),
            const SizedBox(width: 6),
            const Text(
              'days ago',
              style: TextStyle(fontSize: 12, color: AppColors.onSurfaceVariant),
            ),
          ],
        ),
      ],
    );
  }

  Widget _buildWorkModelSelector() {
    final models = [
      {'key': 'all', 'label': 'All Locations'},
      {'key': 'remote_only', 'label': 'Remote Only'},
      {'key': 'onsite_hybrid', 'label': 'On-site / Hybrid'},
    ];

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: models.map((m) {
        final isSelected = _workModel == m['key'];
        return ChoiceChip(
          label: Text(m['label']!),
          selected: isSelected,
          selectedColor: AppColors.primary,
          labelStyle: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          onSelected: (selected) {
            if (selected) {
              setState(() => _workModel = m['key']!);
            }
          },
        );
      }).toList(),
    );
  }

  Widget _buildViewStatusSelector() {
    final views = [
      {'key': 'all', 'label': 'All (Viewed & Unviewed)'},
      {'key': 'unviewed', 'label': 'Unviewed Only'},
      {'key': 'viewed', 'label': 'Viewed Only'},
    ];

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: views.map((v) {
        final isSelected = _viewMode == v['key'];
        return ChoiceChip(
          label: Text(v['label']!),
          selected: isSelected,
          selectedColor: AppColors.primary,
          labelStyle: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          onSelected: (selected) {
            if (selected) {
              setState(() => _viewMode = v['key']!);
            }
          },
        );
      }).toList(),
    );
  }

  Widget _buildApplicationStatusSelector() {
    final statuses = [
      {'key': 'all', 'label': 'All'},
      {'key': 'unapplied', 'label': 'Unapplied'},
      {'key': 'bookmarked', 'label': 'Saved / Bookmarked'},
      {'key': 'applied', 'label': 'Applied'},
    ];

    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: statuses.map((st) {
        final isSelected = _applicationStatus == st['key'];
        return ChoiceChip(
          label: Text(st['label']!),
          selected: isSelected,
          selectedColor: AppColors.primary,
          labelStyle: TextStyle(
            fontSize: 12,
            fontWeight: FontWeight.w600,
            color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
          ),
          onSelected: (selected) {
            if (selected) {
              setState(() => _applicationStatus = st['key']!);
            }
          },
        );
      }).toList(),
    );
  }

  Widget _buildFooterActions() {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 12),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          TextButton.icon(
            onPressed: _handleReset,
            icon: const Icon(Icons.refresh, size: 16),
            label: const Text('Reset All Defaults'),
            style: TextButton.styleFrom(
              foregroundColor: AppColors.onSurfaceVariant,
            ),
          ),
          ElevatedButton(
            onPressed: _handleApply,
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 12),
              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
            ),
            child: const Text('Apply Filters', style: TextStyle(fontWeight: FontWeight.w700)),
          ),
        ],
      ),
    );
  }
}
