import 'dart:convert';
import 'main.dart' show AppColors;
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';

void main() {
  runApp(const SetPreferencesApp());
}

class PreferenceSummary {
  const PreferenceSummary({
    required this.industries,
    required this.targetRoles,
    required this.baseSalary,
    required this.equityExpectation,
  });

  final List<String> industries;
  final List<String> targetRoles;
  final double baseSalary;
  final String equityExpectation;

  String get salaryLabel => '\$${baseSalary.toInt()}k+';

  Map<String, dynamic> toJson() {
    return {
      'industries': industries,
      'targetRoles': targetRoles,
      'baseSalary': baseSalary,
      'equityExpectation': equityExpectation,
    };
  }

  static PreferenceSummary fromJson(Map<String, dynamic> json) {
    return PreferenceSummary(
      industries: List<String>.from(json['industries'] as List<dynamic>? ?? const []),
      targetRoles: List<String>.from(json['targetRoles'] as List<dynamic>? ?? const []),
      baseSalary: (json['baseSalary'] as num?)?.toDouble() ?? 0,
      equityExpectation: json['equityExpectation'] as String? ?? '',
    );
  }

  static const String storageKey = 'job_cruiser.preference_summary';

  static Future<PreferenceSummary?> load() async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(storageKey);
    if (raw == null || raw.isEmpty) {
      return null;
    }

    return PreferenceSummary.fromJson(jsonDecode(raw) as Map<String, dynamic>);
  }

  Future<void> save() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(storageKey, jsonEncode(toJson()));
  }
}

class SetPreferencesApp extends StatelessWidget {
  const SetPreferencesApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Set Preferences',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.background,
        useMaterial3: true,
        fontFamily: 'Inter', // Defaulting to Inter for body/headlines
        colorScheme: const ColorScheme.light(
          primary: AppColors.primary,
          surface: AppColors.surface,
          onSurface: AppColors.onSurface,
        ),
      ),
      home: const SetPreferencesScreen(),
    );
  }
}

class SetPreferencesScreen extends StatefulWidget {
  const SetPreferencesScreen({super.key, this.initialPreferences});

  final PreferenceSummary? initialPreferences;

  @override
  State<SetPreferencesScreen> createState() => _SetPreferencesScreenState();
}

class _SetPreferencesScreenState extends State<SetPreferencesScreen> {
  late final Set<String> _selectedIndustries;
  late final List<String> _currentTargets;
  late final TextEditingController _roleController;
  late double _baseSalary;
  late String _equityExpectation;

  final List<String> _allIndustries = [
    'Fintech',
    'Enterprise SaaS',
    'AI / ML',
    'Healthtech',
    'E-commerce',
    'Cybersecurity',
  ];

  @override
  void initState() {
    super.initState();
    _selectedIndustries = widget.initialPreferences == null
        ? {'Fintech', 'Enterprise SaaS'}
        : widget.initialPreferences!.industries.toSet();
    _currentTargets = widget.initialPreferences == null
        ? ['Director of Product Marketing', 'VP Marketing']
        : List<String>.from(widget.initialPreferences!.targetRoles);
    _roleController = TextEditingController();
    _baseSalary = widget.initialPreferences?.baseSalary ?? 180.0;
    _equityExpectation =
        widget.initialPreferences?.equityExpectation ?? 'Meaningful';

    if (widget.initialPreferences == null) {
      _loadSavedPreferences();
    }
  }

  Future<void> _loadSavedPreferences() async {
    final savedPreferences = await PreferenceSummary.load();
    if (!mounted || savedPreferences == null || widget.initialPreferences != null) {
      return;
    }

    setState(() {
      _selectedIndustries
        ..clear()
        ..addAll(savedPreferences.industries);
      _currentTargets
        ..clear()
        ..addAll(savedPreferences.targetRoles);
      _baseSalary = savedPreferences.baseSalary;
      _equityExpectation = savedPreferences.equityExpectation;
    });
  }

  @override
  void dispose() {
    _roleController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: _buildAppBar(),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20.0, vertical: 24.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _buildHeader(),
            const SizedBox(height: 24),
            _buildTargetIndustries(),
            const SizedBox(height: 24),
            _buildDesiredRoles(),
            const SizedBox(height: 24),
            _buildCompensationTarget(),
            const SizedBox(height: 32),
            _buildSaveButton(),
            const SizedBox(height: 48), // Bottom padding
          ],
        ),
      ),
    );
  }

  PreferredSizeWidget _buildAppBar() {
    return AppBar(
      backgroundColor: AppColors.surface,
      surfaceTintColor: Colors.transparent,
      elevation: 0,
      centerTitle: true,
      leading: IconButton(
        icon: const Icon(Icons.arrow_back, color: AppColors.onSurfaceVariant),
        onPressed: () => Navigator.maybePop(context),
        splashRadius: 24,
      ),
      title: const Text(
        'Preferences',
        style: TextStyle(
          color: AppColors.onSurface,
          fontSize: 20,
          fontWeight: FontWeight.bold,
          letterSpacing: -0.01,
        ),
      ),
      bottom: PreferredSize(
        preferredSize: const Size.fromHeight(1),
        child: Container(
          color: AppColors.outlineVariant.withValues(alpha: 0.5),
          height: 1,
        ),
      ),
    );
  }

  Widget _buildHeader() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Set Preferences',
          style: TextStyle(
            fontSize: 32,
            fontWeight: FontWeight.bold,
            letterSpacing: -0.02,
            color: AppColors.primary,
            height: 1.2,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          'Fine-tune your criteria to receive higher-compatibility role matches.',
          style: TextStyle(
            fontSize: 14,
            color: AppColors.onSurfaceVariant,
            height: 1.4,
          ),
        ),
      ],
    );
  }

  Widget _buildTargetIndustries() {
    return _buildSectionCard(
      icon: Icons.domain,
      title: 'Target Industries',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Select up to 5 priority sectors.',
            style: TextStyle(
              fontSize: 14,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 16),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: _allIndustries.map((industry) {
              final isSelected = _selectedIndustries.contains(industry);
              return GestureDetector(
                onTap: () {
                  setState(() {
                    if (isSelected) {
                      _selectedIndustries.remove(industry);
                    } else {
                      if (_selectedIndustries.length < 5) {
                        _selectedIndustries.add(industry);
                      }
                    }
                  });
                },
                child: AnimatedContainer(
                  duration: const Duration(milliseconds: 200),
                  padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                  decoration: BoxDecoration(
                    color: isSelected ? AppColors.slate900 : AppColors.surface,
                    border: Border.all(
                      color: isSelected ? AppColors.slate900 : AppColors.outlineVariant,
                    ),
                    borderRadius: BorderRadius.circular(999),
                  ),
                  child: Text(
                    industry,
                    style: TextStyle(
                      fontSize: 14,
                      color: isSelected ? Colors.white : AppColors.secondary,
                      fontWeight: FontWeight.w400,
                    ),
                  ),
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }

  Widget _buildDesiredRoles() {
    return _buildSectionCard(
      icon: Icons.work,
      title: 'Desired Roles',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'ADD ROLE',
            style: TextStyle(
              fontFamily: 'Geist',
              fontSize: 12,
              fontWeight: FontWeight.w500,
              letterSpacing: 0.5,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Container(
            decoration: BoxDecoration(
              color: AppColors.surface,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
            ),
            child: TextField(
              controller: _roleController,
              decoration: InputDecoration(
                hintText: 'e.g. Director of Product Marketing',
                hintStyle: const TextStyle(color: AppColors.outline, fontSize: 14),
                prefixIcon: const Icon(Icons.search, color: AppColors.outline),
                suffixIcon: Padding(
                  padding: const EdgeInsets.all(6.0),
                  child: InkWell(
                    onTap: () {
                      if (_roleController.text.isNotEmpty) {
                        setState(() {
                          _currentTargets.add(_roleController.text);
                          _roleController.clear();
                        });
                      }
                    },
                    child: Container(
                      padding: const EdgeInsets.all(6),
                      decoration: BoxDecoration(
                        color: AppColors.successGreen,
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: const Icon(Icons.add, color: Colors.white, size: 20),
                    ),
                  ),
                ),
                border: InputBorder.none,
                contentPadding: const EdgeInsets.symmetric(vertical: 14),
              ),
            ),
          ),
          const SizedBox(height: 16),
          const Text(
            'CURRENT TARGETS',
            style: TextStyle(
              fontFamily: 'Geist',
              fontSize: 12,
              fontWeight: FontWeight.w500,
              letterSpacing: 0.5,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: _currentTargets.map((target) {
              return Container(
                padding: const EdgeInsets.only(left: 12, right: 8, top: 6, bottom: 6),
                decoration: BoxDecoration(
                  color: AppColors.surfaceContainerLow,
                  border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.3)),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Row(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Text(
                      target,
                      style: const TextStyle(
                        fontSize: 14,
                        color: AppColors.onSurface,
                      ),
                    ),
                    const SizedBox(width: 4),
                    GestureDetector(
                      onTap: () {
                        setState(() {
                          _currentTargets.remove(target);
                        });
                      },
                      child: const Icon(
                        Icons.close,
                        size: 16,
                        color: AppColors.outline,
                      ),
                    ),
                  ],
                ),
              );
            }).toList(),
          ),
        ],
      ),
    );
  }

  Widget _buildCompensationTarget() {
    return _buildSectionCard(
      icon: Icons.payments,
      title: 'Compensation Target',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              const Text(
                'BASE SALARY (USD)',
                style: TextStyle(
                  fontFamily: 'Geist',
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  letterSpacing: 0.5,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Text(
                '\$${_baseSalary.toInt()}k+',
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.bold,
                  color: AppColors.primary,
                ),
              ),
            ],
          ),
          const SizedBox(height: 8),
          SliderTheme(
            data: SliderThemeData(
              activeTrackColor: AppColors.successGreen,
              inactiveTrackColor: AppColors.sliderInactive,
              thumbColor: AppColors.successGreen,
              overlayColor: AppColors.successGreen.withValues(alpha: 0.2),
              trackHeight: 4,
              thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 8),
              overlayShape: const RoundSliderOverlayShape(overlayRadius: 16),
            ),
            child: Slider(
              value: _baseSalary,
              min: 100,
              max: 400,
              divisions: 30,
              onChanged: (value) {
                setState(() {
                  _baseSalary = value;
                });
              },
            ),
          ),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: const [
              Text('\$100k', style: TextStyle(fontSize: 11, color: AppColors.outline)),
              Text('\$400k+', style: TextStyle(fontSize: 11, color: AppColors.outline)),
            ],
          ),
          const SizedBox(height: 24),
          const Text(
            'EQUITY EXPECTATION',
            style: TextStyle(
              fontFamily: 'Geist',
              fontSize: 12,
              fontWeight: FontWeight.w500,
              letterSpacing: 0.5,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              _buildEquityOption('Standard'),
              const SizedBox(width: 8),
              _buildEquityOption('Meaningful'),
              const SizedBox(width: 8),
              _buildEquityOption('Founder Level'),
            ],
          ),
        ],
      ),
    );
  }

  Widget _buildEquityOption(String label) {
    final isSelected = _equityExpectation == label;
    return Expanded(
      child: GestureDetector(
        onTap: () {
          setState(() {
            _equityExpectation = label;
          });
        },
        child: AnimatedContainer(
          duration: const Duration(milliseconds: 200),
          padding: const EdgeInsets.symmetric(vertical: 8),
          decoration: BoxDecoration(
            color: isSelected ? AppColors.slate900 : AppColors.surface,
            border: Border.all(
              color: isSelected ? AppColors.slate900 : AppColors.outlineVariant,
            ),
            borderRadius: BorderRadius.circular(6),
          ),
          alignment: Alignment.center,
          child: Text(
            label,
            style: TextStyle(
              fontSize: 14,
              color: isSelected ? Colors.white : AppColors.secondary,
              fontWeight: FontWeight.w400,
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildSectionCard({
    required IconData icon,
    required String title,
    required Widget child,
  }) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.3)),
        boxShadow: [
          BoxShadow(
            color: const Color(0xFF0F172A).withValues(alpha: 0.02),
            blurRadius: 12,
            offset: const Offset(0, 4),
          ),
        ],
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(icon, color: AppColors.secondary, size: 24),
              const SizedBox(width: 8),
              Text(
                title,
                style: const TextStyle(
                  fontSize: 20,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurface,
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }

  Widget _buildSaveButton() {
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton(
        onPressed: () {
          final summary = PreferenceSummary(
              industries: _selectedIndustries.toList()..sort(),
              targetRoles: List<String>.from(_currentTargets),
              baseSalary: _baseSalary,
              equityExpectation: _equityExpectation,
            );

          summary.save();
          Navigator.pop(context, summary);
        },
        style: ElevatedButton.styleFrom(
          backgroundColor: AppColors.successGreen,
          foregroundColor: Colors.white,
          padding: const EdgeInsets.symmetric(vertical: 16),
          shape: RoundedRectangleBorder(
            borderRadius: BorderRadius.circular(8),
          ),
          elevation: 4,
          shadowColor: AppColors.successGreen.withValues(alpha: 0.4),
        ),
        child: Row(
          mainAxisAlignment: MainAxisAlignment.center,
          children: const [
            Text(
              'Save Preferences',
              style: TextStyle(
                fontSize: 20,
                fontWeight: FontWeight.w600,
                letterSpacing: -0.01,
              ),
            ),
            SizedBox(width: 8),
            Icon(Icons.check_circle, size: 24),
          ],
        ),
      ),
    );
  }
}