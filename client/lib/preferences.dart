import 'dart:convert';
import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:syncfusion_flutter_pdf/pdf.dart';
import 'package:url_launcher/url_launcher.dart';
import 'auth.dart';
import 'main.dart' show AppColors;
import 'services/api_service.dart';

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
  late final TextEditingController _locationController;
  late final TextEditingController _emailController;
  late final TextEditingController _phoneController;
  late final TextEditingController _githubController;
  late final TextEditingController _linkedinController;
  late final TextEditingController _portfolioController;
  late final TextEditingController _overleafUrlController;
  late final TextEditingController _overleafSecretController;
  late final TextEditingController _overleafProjectController;
  late final TextEditingController _resumeTemplateController;
  late final TextEditingController _coverLetterTemplateController;
  late final TextEditingController _bioTextController;

  final List<Map<String, TextEditingController>> _customLinkControllers = [];

  bool _isParsingCV = false;
  late double _baseSalary;
  late String _equityExpectation;
  final Set<String> _selectedWorkModels = {'remote', 'hybrid'};

  bool _anyWorkModel = false;
  bool _anyLocation = false;
  bool _hasConfiguredSecret = false;
  bool _obscureSecret = true;

  final List<String> _allIndustries = [
    'Fintech',
    'Enterprise SaaS',
    'AI / ML',
    'Healthtech',
    'E-commerce',
    'Cybersecurity',
    'Edtech',
    'Consumer Tech',
    'Cloud / DevOps',
    'Web3 / Crypto',
    'Gaming',
    'Hardware / IoT',
    'Biotech',
    'Media & Entertainment',
    'Logistics & Supply Chain',
    'Aerospace',
  ];

  final List<String> _popularRoleSuggestions = [
    'Backend Engineer',
    'Fullstack SDE',
    'Frontend Engineer',
    'DevOps / SRE',
    'Data Engineer',
    'Machine Learning Engineer',
    'Mobile Engineer (iOS/Android)',
    'Engineering Manager',
    'Product Manager',
    'System Architect',
    'Embedded Systems Engineer',
    'Security Engineer',
    'QA / Automation Engineer',
  ];

  final List<String> _availableLocations = [
    'India (On-site & Hybrid)',
    'India (Remote)',
    'Global Remote',
    'US / North America Remote',
    'Europe Remote',
  ];
  late Set<String> _selectedLocations;
  String _currency = 'USD';
  int _targetResumePages = 1;
  int _targetCoverLetterPages = 1;
  bool _matchThresholdNotificationEnabled = false;
  int _matchThresholdPercentage = 80;

  @override
  void initState() {
    super.initState();
    _selectedLocations = {'India (On-site & Hybrid)', 'India (Remote)', 'Global Remote'};
    _selectedIndustries = widget.initialPreferences == null
        ? {'Fintech', 'Enterprise SaaS', 'AI / ML'}
        : widget.initialPreferences!.industries.toSet();
    _currentTargets = widget.initialPreferences == null
        ? ['Backend Engineer', 'Fullstack SDE']
        : List<String>.from(widget.initialPreferences!.targetRoles);
    _roleController = TextEditingController();
    _locationController = TextEditingController();
    _emailController = TextEditingController();
    _phoneController = TextEditingController();
    _githubController = TextEditingController();
    _linkedinController = TextEditingController();
    _portfolioController = TextEditingController();
    _overleafUrlController = TextEditingController();
    _overleafSecretController = TextEditingController();
    _overleafProjectController = TextEditingController(text: 'job_applications');
    _resumeTemplateController = TextEditingController(text: 'templates/resume.tex');
    _coverLetterTemplateController = TextEditingController(text: 'templates/cover_letter.tex');
    _bioTextController = TextEditingController();
    _baseSalary = widget.initialPreferences?.baseSalary ?? 120.0;
    _equityExpectation =
        widget.initialPreferences?.equityExpectation ?? 'Meaningful';

    _loadSavedPreferences();
  }

  void _addCustomLink({String label = '', String url = ''}) {
    setState(() {
      _customLinkControllers.add({
        'label': TextEditingController(text: label),
        'url': TextEditingController(text: url),
      });
    });
  }

  void _removeCustomLink(int index) {
    setState(() {
      _customLinkControllers[index]['label']?.dispose();
      _customLinkControllers[index]['url']?.dispose();
      _customLinkControllers.removeAt(index);
    });
  }

  Future<void> _loadSavedPreferences() async {
    final apiPref = await ApiService().fetchPreferences();
    if (!mounted) return;

    if (apiPref != null) {
      setState(() {
        if (apiPref['email'] != null && (apiPref['email'] as String).isNotEmpty) {
          _emailController.text = apiPref['email'] as String;
        }
        if (apiPref['phone'] != null && (apiPref['phone'] as String).isNotEmpty) {
          _phoneController.text = apiPref['phone'] as String;
        }
        if (apiPref['location'] != null && (apiPref['location'] as String).isNotEmpty) {
          _locationController.text = apiPref['location'] as String;
        }
        if (apiPref['github_url'] != null && (apiPref['github_url'] as String).isNotEmpty) {
          _githubController.text = apiPref['github_url'] as String;
        }
        if (apiPref['linkedin_url'] != null && (apiPref['linkedin_url'] as String).isNotEmpty) {
          _linkedinController.text = apiPref['linkedin_url'] as String;
        }
        if (apiPref['portfolio_url'] != null && (apiPref['portfolio_url'] as String).isNotEmpty) {
          _portfolioController.text = apiPref['portfolio_url'] as String;
        }
        if (apiPref['custom_links'] != null && apiPref['custom_links'] is List) {
          for (final linkEntry in _customLinkControllers) {
            linkEntry['label']?.dispose();
            linkEntry['url']?.dispose();
          }
          _customLinkControllers.clear();
          for (final rawItem in (apiPref['custom_links'] as List)) {
            if (rawItem is Map) {
              _customLinkControllers.add({
                'label': TextEditingController(text: rawItem['label']?.toString() ?? ''),
                'url': TextEditingController(text: rawItem['url']?.toString() ?? ''),
              });
            }
          }
        }
        if (apiPref['currency'] != null && (apiPref['currency'] as String).isNotEmpty) {
          _currency = apiPref['currency'] as String;
        }
        if (apiPref['target_roles'] != null && (apiPref['target_roles'] as List).isNotEmpty) {
          _currentTargets
            ..clear()
            ..addAll(List<String>.from(apiPref['target_roles'] as List));
        }
        if (apiPref['target_industries'] != null && (apiPref['target_industries'] as List).isNotEmpty) {
          _selectedIndustries
            ..clear()
            ..addAll(List<String>.from(apiPref['target_industries'] as List));
        }
        if (apiPref['target_locations'] != null && (apiPref['target_locations'] as List).isNotEmpty) {
          _selectedLocations
            ..clear()
            ..addAll(List<String>.from(apiPref['target_locations'] as List));
        }
        if (apiPref['min_salary'] != null && (apiPref['min_salary'] as num) > 0) {
          final num val = apiPref['min_salary'] as num;
          if (_currency == 'INR') {
            _baseSalary = (val.toDouble() / 100000).clamp(0.0, 100.0);
          } else {
            _baseSalary = (val.toDouble() / 1000).clamp(0.0, 400.0);
          }
        } else {
          _baseSalary = 0.0;
        }
        if (apiPref['work_models'] != null && (apiPref['work_models'] as List).isNotEmpty) {
          _selectedWorkModels
            ..clear()
            ..addAll(List<String>.from(apiPref['work_models'] as List));
        }
        if (apiPref['bio_experience_text'] != null && (apiPref['bio_experience_text'] as String).isNotEmpty) {
          _bioTextController.text = apiPref['bio_experience_text'] as String;
        } else if (apiPref['master_cv_text'] != null && (apiPref['master_cv_text'] as String).isNotEmpty) {
          _bioTextController.text = apiPref['master_cv_text'] as String;
        }
        if (apiPref['target_resume_pages'] != null) {
          _targetResumePages = (apiPref['target_resume_pages'] as num).toInt().clamp(1, 4);
        }
        if (apiPref['target_cover_letter_pages'] != null) {
          _targetCoverLetterPages = (apiPref['target_cover_letter_pages'] as num).toInt().clamp(1, 4);
        }
        if (apiPref['match_threshold_notification_enabled'] != null) {
          _matchThresholdNotificationEnabled = apiPref['match_threshold_notification_enabled'] == true;
        }
        if (apiPref['match_threshold_percentage'] != null && (apiPref['match_threshold_percentage'] as num) > 0) {
          _matchThresholdPercentage = (apiPref['match_threshold_percentage'] as num).toInt().clamp(50, 100);
        }
      });
    }

    final overleaf = await ApiService().fetchOverleafConfig();
    if (overleaf != null && mounted) {
      setState(() {
        _overleafUrlController.text = overleaf['deployment_url'] ?? '';
        _overleafProjectController.text = overleaf['project_name'] ?? 'job_applications';
        if (overleaf['resume_template_path'] != null && (overleaf['resume_template_path'] as String).isNotEmpty) {
          _resumeTemplateController.text = overleaf['resume_template_path'] as String;
        }
        if (overleaf['cover_letter_template_path'] != null && (overleaf['cover_letter_template_path'] as String).isNotEmpty) {
          _coverLetterTemplateController.text = overleaf['cover_letter_template_path'] as String;
        }
        _hasConfiguredSecret = overleaf['has_secret'] == true;
        if (overleaf['mcp_secret'] != null && (overleaf['mcp_secret'] as String).isNotEmpty) {
          _overleafSecretController.text = overleaf['mcp_secret'] as String;
        }
      });
    }
  }

  @override
  void dispose() {
    _roleController.dispose();
    _locationController.dispose();
    _emailController.dispose();
    _phoneController.dispose();
    _githubController.dispose();
    _linkedinController.dispose();
    _portfolioController.dispose();
    _overleafUrlController.dispose();
    _overleafSecretController.dispose();
    _overleafProjectController.dispose();
    _resumeTemplateController.dispose();
    _coverLetterTemplateController.dispose();
    _bioTextController.dispose();
    for (final linkEntry in _customLinkControllers) {
      linkEntry['label']?.dispose();
      linkEntry['url']?.dispose();
    }
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
            _buildContactAndLinksCard(),
            const SizedBox(height: 24),
            _buildExperienceCard(),
            const SizedBox(height: 24),
            _buildDesiredRoles(),
            const SizedBox(height: 24),
            _buildTargetIndustries(),
            const SizedBox(height: 24),
            _buildTargetLocations(),
            const SizedBox(height: 24),
            _buildWorkModels(),
            const SizedBox(height: 24),
            _buildCompensationTarget(),
            const SizedBox(height: 24),
            _buildMatchNotificationCard(),
            const SizedBox(height: 24),
            _buildOverleafCard(),
            const SizedBox(height: 24),
            _buildPageBudgetCard(),
            const SizedBox(height: 32),
            _buildSaveButton(),
            const SizedBox(height: 48),
          ],
        ),
      ),
    );
  }

  Widget _buildTargetLocations() {
    return _buildSectionCard(
      icon: Icons.location_on,
      title: 'Target Locations',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'ANY LOCATION / NO PREFERENCE',
                style: TextStyle(
                  fontFamily: 'Geist',
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Switch(
                value: _anyLocation,
                onChanged: (val) {
                  setState(() {
                    _anyLocation = val;
                  });
                },
              ),
            ],
          ),
          if (!_anyLocation) ...[
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _availableLocations.map((loc) {
                final isSelected = _selectedLocations.contains(loc);
                return FilterChip(
                  label: Text(loc),
                  selected: isSelected,
                  selectedColor: AppColors.primary.withValues(alpha: 0.15),
                  checkmarkColor: AppColors.primary,
                  onSelected: (val) {
                    setState(() {
                      if (val) {
                        _selectedLocations.add(loc);
                      } else {
                        _selectedLocations.remove(loc);
                      }
                    });
                  },
                );
              }).toList(),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildWorkModels() {
    final models = [
      {'key': 'remote', 'label': 'Remote'},
      {'key': 'hybrid', 'label': 'Hybrid'},
      {'key': 'onsite', 'label': 'On-site'},
    ];

    return _buildSectionCard(
      icon: Icons.home_work,
      title: 'Preferred Work Models',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'ANY WORK MODEL / NO PREFERENCE',
                style: TextStyle(
                  fontFamily: 'Geist',
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Switch(
                value: _anyWorkModel,
                onChanged: (val) {
                  setState(() {
                    _anyWorkModel = val;
                  });
                },
              ),
            ],
          ),
          if (!_anyWorkModel) ...[
            const SizedBox(height: 12),
            Wrap(
              spacing: 8,
              children: models.map((m) {
                final isSelected = _selectedWorkModels.contains(m['key']);
                return FilterChip(
                  label: Text(m['label']!),
                  selected: isSelected,
                  selectedColor: AppColors.primary.withValues(alpha: 0.15),
                  checkmarkColor: AppColors.primary,
                  onSelected: (val) {
                    setState(() {
                      if (val) {
                        _selectedWorkModels.add(m['key']!);
                      } else {
                        _selectedWorkModels.remove(m['key']!);
                      }
                    });
                  },
                );
              }).toList(),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildOverleafCard() {
    return _buildSectionCard(
      icon: Icons.description,
      title: 'Open-Overleaf TeX Sync & Format Guidance',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Self-hosted Open-Overleaf server for automated LaTeX resume & cover letter compilation.',
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _overleafUrlController,
            decoration: const InputDecoration(
              labelText: 'Open-Overleaf Server URL',
              hintText: 'e.g. https://overleaf.example.com',
              border: OutlineInputBorder(),
              helperText: 'Base URL of your running Open-Overleaf instance or MCP endpoint',
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _overleafSecretController,
            obscureText: _obscureSecret,
            decoration: InputDecoration(
              labelText: _hasConfiguredSecret ? 'MCP Secret / Access Token' : 'MCP Secret / Access Token (Required)',
              hintText: _hasConfiguredSecret ? '•••••••••••••••• (Configured & Encrypted)' : 'e.g. OVERLEAF_MCP_SECRET or OVERLEAF_MCP_TOKEN',
              border: const OutlineInputBorder(),
              helperText: _hasConfiguredSecret
                  ? 'Access token is securely encrypted (AES-256-GCM). Enter a new value to update.'
                  : 'Required for private authentication. Encrypted with AES-256 at rest.',
              suffixIcon: IconButton(
                icon: Icon(_obscureSecret ? Icons.visibility_off : Icons.visibility),
                onPressed: () => setState(() => _obscureSecret = !_obscureSecret),
              ),
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _overleafProjectController,
            decoration: const InputDecoration(
              labelText: 'Project / Workspace Name',
              hintText: 'job_applications',
              border: OutlineInputBorder(),
              helperText: 'Top-level folder or workspace in Open-Overleaf (defaults to job_applications)',
            ),
          ),
          const SizedBox(height: 20),
          const Divider(),
          const SizedBox(height: 12),
          Row(
            children: const [
              Icon(Icons.palette_outlined, size: 18, color: AppColors.primary),
              SizedBox(width: 8),
              Text(
                'Baseline Format & Style Guidance',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
              ),
            ],
          ),
          const SizedBox(height: 6),
          const Text(
            'The AI uses these LaTeX templates as visual & structural blueprints (fonts, margins, macro structures) when tailoring for each job.',
            style: TextStyle(fontSize: 12, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _resumeTemplateController,
            decoration: const InputDecoration(
              labelText: 'Resume Baseline Template Path',
              hintText: 'templates/resume.tex',
              border: OutlineInputBorder(),
              helperText: 'Path to baseline resume .tex inside your Overleaf project',
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _coverLetterTemplateController,
            decoration: const InputDecoration(
              labelText: 'Cover Letter Baseline Template Path',
              hintText: 'templates/cover_letter.tex',
              border: OutlineInputBorder(),
              helperText: 'Path to baseline cover letter .tex inside your Overleaf project',
            ),
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.open_in_browser, size: 16),
                  label: const Text('Edit in Overleaf'),
                  onPressed: () async {
                    final rawUrl = _overleafUrlController.text.trim();
                    if (rawUrl.isEmpty) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text('Please configure your Open-Overleaf URL first')),
                      );
                      return;
                    }
                    final project = _overleafProjectController.text.trim().isEmpty ? 'job_applications' : _overleafProjectController.text.trim();
                    final uri = Uri.parse('${rawUrl.replaceAll(RegExp(r"/+$"), "")}/?project=$project');
                    if (await canLaunchUrl(uri)) {
                      await launchUrl(uri, mode: LaunchMode.externalApplication);
                    } else {
                      if (!mounted) return;
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text('Could not launch: $uri')),
                      );
                    }
                  },
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.refresh, size: 16),
                  label: const Text('Reset Defaults'),
                  onPressed: () async {
                    final ok = await ApiService().seedDefaultTemplates();
                    if (!mounted) return;
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text(
                          ok
                              ? 'Default resume and cover letter formats restored in Open-Overleaf!'
                              : 'Failed to restore default formats. Check server connection.',
                        ),
                      ),
                    );
                  },
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: ElevatedButton.icon(
              icon: const Icon(Icons.save, size: 18),
              label: const Text('Save Overleaf & Format Settings'),
              onPressed: () async {
                final url = _overleafUrlController.text.trim();
                final secret = _overleafSecretController.text.trim();
                final project = _overleafProjectController.text.trim();
                final resumeTemplate = _resumeTemplateController.text.trim();
                final coverLetterTemplate = _coverLetterTemplateController.text.trim();

                if (url.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Please enter your Open-Overleaf Server URL')),
                  );
                  return;
                }

                if (!_hasConfiguredSecret && secret.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Please provide your MCP Secret / Access Token to secure your Open-Overleaf connection')),
                  );
                  return;
                }

                final ok = await ApiService().saveOverleafConfig(
                  deploymentUrl: url,
                  mcpSecret: secret.isNotEmpty ? secret : null,
                  projectName: project.isNotEmpty ? project : 'job_applications',
                  resumeTemplatePath: resumeTemplate.isNotEmpty ? resumeTemplate : 'templates/resume.tex',
                  coverLetterTemplatePath: coverLetterTemplate.isNotEmpty ? coverLetterTemplate : 'templates/cover_letter.tex',
                );

                if (!mounted) return;
                if (ok && secret.isNotEmpty) {
                  setState(() => _hasConfiguredSecret = true);
                  _overleafSecretController.clear();
                }
                ScaffoldMessenger.of(context).showSnackBar(
                  SnackBar(
                    content: Text(
                      ok
                          ? 'Open-Overleaf configuration and format settings saved successfully!'
                          : 'Failed to save Open-Overleaf configuration',
                    ),
                  ),
                );
              },
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildPageBudgetCard() {
    return _buildSectionCard(
      icon: Icons.auto_stories,
      title: 'Target Page Limits',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Configure strict page budgets. The AI tailoring engine applies multi-pass tightening loops to fit your content cleanly across the full page without leaving empty white space.',
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          const Text(
            'Resume Target Length',
            style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: AppColors.primary),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [1, 2, 3, 4].map((page) {
              final isSelected = _targetResumePages == page;
              return ChoiceChip(
                label: Text('$page ${page == 1 ? 'Page' : 'Pages'}'),
                selected: isSelected,
                selectedColor: AppColors.primary.withValues(alpha: 0.15),
                checkmarkColor: AppColors.primary,
                onSelected: (selected) {
                  if (selected) setState(() => _targetResumePages = page);
                },
              );
            }).toList(),
          ),
          const SizedBox(height: 16),
          const Text(
            'Cover Letter Target Length',
            style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: AppColors.primary),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [1, 2, 3, 4].map((page) {
              final isSelected = _targetCoverLetterPages == page;
              return ChoiceChip(
                label: Text('$page ${page == 1 ? 'Page' : 'Pages'}'),
                selected: isSelected,
                selectedColor: AppColors.primary.withValues(alpha: 0.15),
                checkmarkColor: AppColors.primary,
                onSelected: (selected) {
                  if (selected) setState(() => _targetCoverLetterPages = page);
                },
              );
            }).toList(),
          ),
        ],
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
      actions: [
        IconButton(
          icon: const Icon(Icons.logout, color: AppColors.error),
          tooltip: 'Sign Out',
          onPressed: () async {
            await ApiService().clearToken();
            if (!mounted) return;
            Navigator.pushAndRemoveUntil(
              context,
              MaterialPageRoute(builder: (_) => const AuthScreen()),
              (route) => false,
            );
          },
        ),
      ],
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

  Future<void> _extractPdfToBio() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: ['pdf', 'txt'],
        withData: true,
      );

      if (result == null || result.files.isEmpty) {
        return;
      }

      final file = result.files.first;

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Reading ${file.name}...')),
      );

      String extractedText = '';
      if (file.bytes != null) {
        if (file.name.toLowerCase().endsWith('.pdf')) {
          final PdfDocument document = PdfDocument(inputBytes: file.bytes!);
          extractedText = PdfTextExtractor(document).extractText();
          document.dispose();
        } else {
          extractedText = utf8.decode(file.bytes!, allowMalformed: true);
        }

        final cleanLines = extractedText
            .split('\n')
            .map((line) => line.trim())
            .where((line) => line.isNotEmpty && !line.startsWith('%PDF-'))
            .toList();
        extractedText = cleanLines.join('\n');
      }

      if (extractedText.trim().isEmpty) {
        extractedText = "Parsed experience summary from ${file.name}";
      }

      setState(() {
        _isParsingCV = true;
      });

      final parsed = await ApiService().parseCVWithGemini(extractedText.trim());

      if (!mounted) return;
      setState(() {
        _isParsingCV = false;
        if (parsed != null && parsed['bio_summary'] != null && (parsed['bio_summary'] as String).isNotEmpty) {
          _bioTextController.text = parsed['bio_summary'] as String;
        } else {
          _bioTextController.text = extractedText.trim();
        }
      });
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('CV extracted successfully into bio summary!')),
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _isParsingCV = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Error parsing CV file: $e')),
      );
    }
  }

  Widget _buildExperienceCard() {
    return _buildSectionCard(
      icon: Icons.badge,
      title: 'Master CV & Bio Summary',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Extract your experience background automatically from your CV file or enter it manually below.',
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 12),
          SizedBox(
            width: double.infinity,
            child: OutlinedButton.icon(
              icon: _isParsingCV
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.upload_file, size: 18),
              label: Text(_isParsingCV ? 'Extracting text from CV...' : 'Extract Bio from CV (PDF / TXT)'),
              onPressed: _isParsingCV ? null : _extractPdfToBio,
            ),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _bioTextController,
            maxLines: 8,
            decoration: const InputDecoration(
              labelText: 'Master Bio & Experience Text',
              hintText: 'Describe your technical background, skills, key achievements, and project history...',
              border: OutlineInputBorder(),
              alignLabelWithHint: true,
            ),
          ),
        ],
      ),
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
            'QUICK ADD POPULAR ROLES',
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
            runSpacing: 6,
            children: _popularRoleSuggestions.map((role) {
              final isSelected = _currentTargets.contains(role);
              return FilterChip(
                label: Text(role, style: const TextStyle(fontSize: 12)),
                selected: isSelected,
                onSelected: (val) {
                  setState(() {
                    if (val) {
                      if (!_currentTargets.contains(role)) {
                        _currentTargets.add(role);
                      }
                    } else {
                      _currentTargets.remove(role);
                    }
                  });
                },
              );
            }).toList(),
          ),
          const SizedBox(height: 16),
          const Text(
            'ADD CUSTOM ROLE',
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
    final bool isINR = _currency == 'INR';
    final double minVal = 0.0;
    final double maxVal = isINR ? 100.0 : 400.0;
    final double currentVal = _baseSalary.clamp(minVal, maxVal);

    return _buildSectionCard(
      icon: Icons.payments,
      title: 'Compensation Target',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Text(
                'PREFERRED CURRENCY',
                style: TextStyle(
                  fontFamily: 'Geist',
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  letterSpacing: 0.5,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
              SizedBox(
                width: double.infinity,
                child: SegmentedButton<String>(
                  segments: const [
                    ButtonSegment(value: 'USD', label: Text(r'USD ($)')),
                    ButtonSegment(value: 'INR', label: Text(r'INR (₹)')),
                  ],
                  selected: {_currency},
                  onSelectionChanged: (newSelection) {
                    setState(() {
                      _currency = newSelection.first;
                      if (_currency == 'INR' && _baseSalary > 100) {
                        _baseSalary = 20.0;
                      } else if (_currency == 'USD' && _baseSalary < 10) {
                        _baseSalary = 100.0;
                      }
                    });
                  },
                ),
              ),
            ],
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            crossAxisAlignment: CrossAxisAlignment.end,
            children: [
              Text(
                isINR ? 'BASE SALARY (INR)' : 'BASE SALARY (USD)',
                style: const TextStyle(
                  fontFamily: 'Geist',
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  letterSpacing: 0.5,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Text(
                isINR ? '₹${currentVal.toInt()} LPA+' : '\$${currentVal.toInt()}k+',
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
              value: currentVal,
              min: minVal,
              max: maxVal,
              divisions: isINR ? 100 : 80,
              onChanged: (value) {
                setState(() {
                  _baseSalary = value;
                });
              },
            ),
          ),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Text(isINR ? '₹0 LPA' : '\$0k', style: const TextStyle(fontSize: 11, color: AppColors.outline)),
              Text(isINR ? '₹100 LPA+' : '\$400k+', style: const TextStyle(fontSize: 11, color: AppColors.outline)),
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
              Expanded(
                child: Text(
                  title,
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.w600,
                    color: AppColors.onSurface,
                  ),
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

  Widget _buildContactAndLinksCard() {
    return _buildSectionCard(
      icon: Icons.contact_page_outlined,
      title: 'Contact Information & Links',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Populate your authentic links and contact info to be included in tailored LaTeX resumes and cover letters.',
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _emailController,
            keyboardType: TextInputType.emailAddress,
            decoration: const InputDecoration(
              labelText: 'Contact Email',
              hintText: 'e.g. user@example.com',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _phoneController,
                  keyboardType: TextInputType.phone,
                  decoration: const InputDecoration(
                    labelText: 'Phone',
                    hintText: 'e.g. +1 555-0199',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextField(
                  controller: _locationController,
                  decoration: const InputDecoration(
                    labelText: 'Location / City',
                    hintText: 'e.g. Bengaluru, India',
                    border: OutlineInputBorder(),
                    isDense: true,
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _githubController,
            decoration: const InputDecoration(
              labelText: 'GitHub Profile URL',
              hintText: 'https://github.com/username',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _linkedinController,
            decoration: const InputDecoration(
              labelText: 'LinkedIn Profile URL',
              hintText: 'https://linkedin.com/in/username',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _portfolioController,
            decoration: const InputDecoration(
              labelText: 'Portfolio / Personal Website URL',
              hintText: 'https://yourportfolio.dev',
              border: OutlineInputBorder(),
              isDense: true,
            ),
          ),
          const SizedBox(height: 16),
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'Additional Profile Links',
                style: TextStyle(
                  fontSize: 13,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurface,
                ),
              ),
              TextButton.icon(
                onPressed: () => _addCustomLink(),
                icon: const Icon(Icons.add, size: 18),
                label: const Text('Add Link'),
              ),
            ],
          ),
          if (_customLinkControllers.isNotEmpty) ...[
            const SizedBox(height: 8),
            ..._customLinkControllers.asMap().entries.map((entry) {
              final index = entry.key;
              final linkItem = entry.value;
              return Padding(
                padding: const EdgeInsets.only(bottom: 8.0),
                child: Row(
                  children: [
                    SizedBox(
                      width: 110,
                      child: TextField(
                        controller: linkItem['label'],
                        decoration: const InputDecoration(
                          labelText: 'Label',
                          hintText: 'e.g. Blog',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: TextField(
                        controller: linkItem['url'],
                        decoration: const InputDecoration(
                          labelText: 'URL',
                          hintText: 'https://...',
                          border: OutlineInputBorder(),
                          isDense: true,
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close, color: AppColors.error, size: 20),
                      onPressed: () => _removeCustomLink(index),
                      tooltip: 'Remove link',
                    ),
                  ],
                ),
              );
            }),
          ],
        ],
      ),
    );
  }

  Widget _buildMatchNotificationCard() {
    return _buildSectionCard(
      icon: Icons.notifications_active_outlined,
      title: 'Match Notifications',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: const [
                    Text(
                      'NOTIFY ON HIGH MATCHES',
                      style: TextStyle(
                        fontFamily: 'Geist',
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                        color: AppColors.onSurfaceVariant,
                      ),
                    ),
                    SizedBox(height: 4),
                    Text(
                      'Receive in-app alerts and notifications when new job matches meet or exceed your percentage threshold.',
                      style: TextStyle(
                        fontSize: 13,
                        color: AppColors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              Switch(
                value: _matchThresholdNotificationEnabled,
                onChanged: (val) {
                  setState(() {
                    _matchThresholdNotificationEnabled = val;
                  });
                },
              ),
            ],
          ),
          if (_matchThresholdNotificationEnabled) ...[
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  'Minimum Match Threshold',
                  style: TextStyle(
                    fontSize: 14,
                    fontWeight: FontWeight.w600,
                    color: AppColors.onSurface,
                  ),
                ),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.primary.withValues(alpha: 0.1),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: AppColors.primary.withValues(alpha: 0.3)),
                  ),
                  child: Text(
                    '${_matchThresholdPercentage.toInt()}%+',
                    style: const TextStyle(
                      fontWeight: FontWeight.bold,
                      color: AppColors.primary,
                      fontSize: 14,
                    ),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Slider(
              value: _matchThresholdPercentage.toDouble().clamp(50.0, 95.0),
              min: 50.0,
              max: 95.0,
              divisions: 9,
              label: '${_matchThresholdPercentage.toInt()}%',
              onChanged: (val) {
                setState(() {
                  _matchThresholdPercentage = val.round();
                });
              },
            ),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: const [
                Text('50% (Broad)', style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
                Text('80% (Recommended)', style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
                Text('95% (Strict)', style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
              ],
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildSaveButton() {
    return SizedBox(
      width: double.infinity,
      child: ElevatedButton(
        onPressed: () async {
          final summary = PreferenceSummary(
            industries: _selectedIndustries.toList()..sort(),
            targetRoles: List<String>.from(_currentTargets),
            baseSalary: _baseSalary,
            equityExpectation: _equityExpectation,
          );

          await summary.save();

          final apiService = ApiService();
          final profile = await apiService.fetchProfile();
          final fullName = profile?['full_name'] ?? 'User';

          final int rawSalary = _currency == 'INR'
              ? _baseSalary.toInt() * 100000
              : _baseSalary.toInt() * 1000;

          final customLinksPayload = _customLinkControllers
              .map((entry) => {
                    'label': entry['label']?.text.trim() ?? '',
                    'url': entry['url']?.text.trim() ?? '',
                  })
              .where((entry) => entry['url']!.isNotEmpty)
              .toList();

          await apiService.savePreferences({
            'full_name': fullName,
            'email': _emailController.text.trim(),
            'phone': _phoneController.text.trim(),
            'location': _locationController.text.trim(),
            'github_url': _githubController.text.trim(),
            'linkedin_url': _linkedinController.text.trim(),
            'portfolio_url': _portfolioController.text.trim(),
            'custom_links': customLinksPayload,
            'target_roles': _currentTargets,
            'target_industries': _selectedIndustries.toList(),
            'target_locations': _selectedLocations.toList(),
            'work_models': _selectedWorkModels.toList(),
            'min_salary': rawSalary,
            'currency': _currency,
            'bio_experience_text': _bioTextController.text,
            'master_cv_text': _bioTextController.text,
            'target_resume_pages': _targetResumePages,
            'target_cover_letter_pages': _targetCoverLetterPages,
            'match_threshold_notification_enabled': _matchThresholdNotificationEnabled,
            'match_threshold_percentage': _matchThresholdPercentage,
          });

          if (!mounted) return;
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