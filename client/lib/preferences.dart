import "package:flutter/foundation.dart" show kIsWeb;
import "package:flutter/material.dart";
import "package:url_launcher/url_launcher.dart";
import "auth.dart";
import "main.dart" show AppColors;
import "services/api_service.dart";
import "services/app_version_service.dart";
import "screens/edit_profile_screen.dart";

void main() {
  runApp(const SetPreferencesApp());
}

/// Structured summary of user target preferences saved locally and remotely.
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

  String get salaryLabel => "\$${baseSalary.toInt()}k+";

  Map<String, dynamic> toJson() {
    return {
      "industries": industries,
      "targetRoles": targetRoles,
      "baseSalary": baseSalary,
      "equityExpectation": equityExpectation,
    };
  }

  static PreferenceSummary fromJson(Map<String, dynamic> json) {
    return PreferenceSummary(
      industries: List<String>.from(json["industries"] as List<dynamic>? ?? const []),
      targetRoles: List<String>.from(json["targetRoles"] as List<dynamic>? ?? const []),
      baseSalary: (json["baseSalary"] as num?)?.toDouble() ?? 0,
      equityExpectation: json["equityExpectation"] as String? ?? "",
    );
  }
}

/// Standalone entry wrapper for preferences screen.
class SetPreferencesApp extends StatelessWidget {
  const SetPreferencesApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: "Set Preferences",
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        scaffoldBackgroundColor: AppColors.background,
        useMaterial3: true,
        fontFamily: "Inter",
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

/// Dedicated screen for configuring job matching criteria, search filters, compensation targets, and Open-Overleaf formatting.
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
  late final TextEditingController _overleafUrlController;
  late final TextEditingController _overleafSecretController;
  late final TextEditingController _overleafProjectController;
  late final TextEditingController _resumeTemplateController;
  late final TextEditingController _coverLetterTemplateController;

  late double _baseSalary;
  late String _equityExpectation;
  final Set<String> _selectedWorkModels = {};
  AppVersionDetails? _appVersionDetails;

  bool _anyRole = true;
  bool _anyIndustry = true;
  bool _anyLocation = true;
  bool _anyWorkModel = true;
  bool _anySalary = true;
  bool _hasConfiguredSecret = false;
  bool _obscureSecret = true;

  final List<String> _allIndustries = [
    "Fintech",
    "Enterprise SaaS",
    "AI / ML",
    "Healthtech",
    "E-commerce",
    "Cybersecurity",
    "Edtech",
    "Consumer Tech",
    "Cloud / DevOps",
    "Web3 / Crypto",
    "Gaming",
    "Hardware / IoT",
    "Biotech",
    "Media & Entertainment",
    "Logistics & Supply Chain",
    "Aerospace",
  ];

  final List<String> _popularRoleSuggestions = [
    "Backend Engineer",
    "Fullstack SDE",
    "Frontend Engineer",
    "DevOps / SRE",
    "Data Engineer",
    "Machine Learning Engineer",
    "Mobile Engineer (iOS/Android)",
    "Engineering Manager",
    "Product Manager",
    "System Architect",
    "Embedded Systems Engineer",
    "Security Engineer",
    "QA / Automation Engineer",
  ];

  final List<String> _availableLocations = [
    "India (On-site & Hybrid)",
    "India (Remote)",
    "Global Remote",
    "US / North America Remote",
    "Europe Remote",
  ];
  late Set<String> _selectedLocations;
  String _currency = "USD";
  int _targetResumePages = 1;
  int _targetCoverLetterPages = 1;
  bool _matchThresholdNotificationEnabled = false;
  int _matchThresholdPercentage = 80;
  late TextEditingController _notificationCriteriaController;

  @override
  void initState() {
    super.initState();
    _selectedLocations = {};
    _selectedIndustries = {};
    _currentTargets = [];
    _roleController = TextEditingController();
    _overleafUrlController = TextEditingController();
    _overleafSecretController = TextEditingController();
    _overleafProjectController = TextEditingController(text: "job_applications");
    _resumeTemplateController = TextEditingController(text: "templates/resume.tex");
    _coverLetterTemplateController = TextEditingController(text: "templates/cover_letter.tex");
    _notificationCriteriaController = TextEditingController();
    _baseSalary = 0.0;
    _equityExpectation = "";

    _loadSavedPreferences();
    _loadAppVersionDetails();
  }

  Future<void> _loadAppVersionDetails() async {
    const service = AppVersionService();
    final details = await service.getVersionDetails();
    if (!mounted) return;
    setState(() => _appVersionDetails = details);
  }

  Future<void> _loadSavedPreferences() async {
    final apiPref = await ApiService().fetchPreferences();
    if (!mounted) return;

    if (apiPref != null) {
      setState(() {
        if (apiPref["currency"] != null && (apiPref["currency"] as String).isNotEmpty) {
          _currency = apiPref["currency"] as String;
        }
        final loadedRoles = (apiPref["target_roles"] as List? ?? [])
            .map((item) => item.toString())
            .where((role) => role != "Any Role" && role != "All Roles")
            .toList();
        _anyRole = loadedRoles.isEmpty;
        _currentTargets
          ..clear()
          ..addAll(loadedRoles);

        final loadedIndustries = (apiPref["target_industries"] as List? ?? [])
            .map((item) => item.toString())
            .where((industry) => industry != "Any Industry")
            .toList();
        _anyIndustry = loadedIndustries.isEmpty;
        _selectedIndustries
          ..clear()
          ..addAll(loadedIndustries);

        final loadedLocations = (apiPref["target_locations"] as List? ?? [])
            .map((item) => item.toString())
            .toList();
        if (loadedLocations.isEmpty || loadedLocations.contains("Any Location")) {
          _anyLocation = true;
          _selectedLocations.clear();
        } else {
          _anyLocation = false;
          _selectedLocations
            ..clear()
            ..addAll(loadedLocations);
        }

        final num? val = apiPref["min_salary"] as num?;
        if (val != null && val > 0) {
          _anySalary = false;
          if (_currency == "INR") {
            _baseSalary = (val.toDouble() / 100000).clamp(0.0, 100.0);
          } else {
            _baseSalary = (val.toDouble() / 1000).clamp(0.0, 400.0);
          }
        } else {
          _anySalary = true;
          _baseSalary = 0.0;
        }

        final loadedWorkModels = (apiPref["work_models"] as List? ?? [])
            .map((item) => item.toString())
            .toList();
        if (loadedWorkModels.isEmpty || loadedWorkModels.contains("any")) {
          _anyWorkModel = true;
          _selectedWorkModels.clear();
        } else {
          _anyWorkModel = false;
          _selectedWorkModels
            ..clear()
            ..addAll(loadedWorkModels);
        }

        if (apiPref["target_resume_pages"] != null) {
          _targetResumePages = (apiPref["target_resume_pages"] as num).toInt().clamp(1, 4);
        }
        if (apiPref["target_cover_letter_pages"] != null) {
          _targetCoverLetterPages = (apiPref["target_cover_letter_pages"] as num).toInt().clamp(1, 4);
        }
        if (apiPref["match_threshold_notification_enabled"] != null) {
          _matchThresholdNotificationEnabled = apiPref["match_threshold_notification_enabled"] == true;
        }
        if (apiPref["match_threshold_percentage"] != null && (apiPref["match_threshold_percentage"] as num) > 0) {
          _matchThresholdPercentage = (apiPref["match_threshold_percentage"] as num).toInt().clamp(50, 100);
        }
        if (apiPref["notification_prompt_criteria"] != null) {
          _notificationCriteriaController.text = apiPref["notification_prompt_criteria"].toString();
        }
      });
    }

    final overleaf = await ApiService().fetchOverleafConfig();
    if (overleaf != null && mounted) {
      setState(() {
        _overleafUrlController.text = overleaf["deployment_url"] ?? "";
        _overleafProjectController.text = overleaf["project_name"] ?? "job_applications";
        if (overleaf["resume_template_path"] != null && (overleaf["resume_template_path"] as String).isNotEmpty) {
          _resumeTemplateController.text = overleaf["resume_template_path"] as String;
        }
        if (overleaf["cover_letter_template_path"] != null && (overleaf["cover_letter_template_path"] as String).isNotEmpty) {
          _coverLetterTemplateController.text = overleaf["cover_letter_template_path"] as String;
        }
        _hasConfiguredSecret = overleaf["has_secret"] == true;
        if (overleaf["mcp_secret"] != null && (overleaf["mcp_secret"] as String).isNotEmpty) {
          _overleafSecretController.text = overleaf["mcp_secret"] as String;
        }
      });
    }
  }

  @override
  void dispose() {
    _roleController.dispose();
    _overleafUrlController.dispose();
    _overleafSecretController.dispose();
    _overleafProjectController.dispose();
    _resumeTemplateController.dispose();
    _coverLetterTemplateController.dispose();
    _notificationCriteriaController.dispose();
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
            const SizedBox(height: 16),
            _buildProfileEditBanner(),
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
            const SizedBox(height: 24),
            _buildAppVersionCard(),
            const SizedBox(height: 32),
            _buildSaveButton(),
            const SizedBox(height: 24),
            _buildVersionFooter(),
            const SizedBox(height: 48),
          ],
        ),
      ),
    );
  }

  Widget _buildProfileEditBanner() {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.5)),
      ),
      child: Row(
        children: [
          const Icon(Icons.person_outline, size: 20, color: AppColors.primary),
          const SizedBox(width: 12),
          const Expanded(
            child: Text(
              "Looking to update your contact links, bio, skills, or projects?",
              style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
            ),
          ),
          TextButton(
            onPressed: () {
              Navigator.of(context).push(
                MaterialPageRoute(builder: (_) => const EditProfileScreen()),
              );
            },
            child: const Text("Edit Profile"),
          ),
        ],
      ),
    );
  }

  Widget _buildTargetLocations() {
    return _buildSectionCard(
      icon: Icons.location_on,
      title: "Target Locations",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                "ANY LOCATION / NO PREFERENCE",
                style: TextStyle(
                  fontFamily: "Geist",
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
      {"key": "remote", "label": "Remote"},
      {"key": "hybrid", "label": "Hybrid"},
      {"key": "onsite", "label": "On-site"},
    ];

    return _buildSectionCard(
      icon: Icons.business,
      title: "Work Models",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                "ANY WORK MODEL / NO PREFERENCE",
                style: TextStyle(
                  fontFamily: "Geist",
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
                final isSelected = _selectedWorkModels.contains(m["key"]);
                return FilterChip(
                  label: Text(m["label"]!),
                  selected: isSelected,
                  selectedColor: AppColors.primary.withValues(alpha: 0.15),
                  checkmarkColor: AppColors.primary,
                  onSelected: (val) {
                    setState(() {
                      if (val) {
                        _selectedWorkModels.add(m["key"]!);
                      } else {
                        _selectedWorkModels.remove(m["key"]!);
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
      title: "Open-Overleaf TeX Sync & Format Guidance",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            "Self-hosted Open-Overleaf server for automated LaTeX resume & cover letter compilation.",
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _overleafUrlController,
            decoration: const InputDecoration(
              labelText: "Open-Overleaf Server URL",
              hintText: "e.g. https://overleaf.example.com",
              border: OutlineInputBorder(),
              helperText: "Base URL of your running Open-Overleaf instance or MCP endpoint",
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _overleafSecretController,
            obscureText: _obscureSecret,
            decoration: InputDecoration(
              labelText: _hasConfiguredSecret ? "MCP Secret / Access Token" : "MCP Secret / Access Token (Required)",
              hintText: _hasConfiguredSecret ? "•••••••••••••••• (Configured & Encrypted)" : "e.g. OVERLEAF_MCP_SECRET or OVERLEAF_MCP_TOKEN",
              border: const OutlineInputBorder(),
              helperText: _hasConfiguredSecret
                  ? "Access token is securely encrypted (AES-256-GCM). Enter a new value to update."
                  : "Required for private authentication. Encrypted with AES-256 at rest.",
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
              labelText: "Project / Workspace Name",
              hintText: "job_applications",
              border: OutlineInputBorder(),
              helperText: "Top-level folder or workspace in Open-Overleaf (defaults to job_applications)",
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
                "Baseline Format & Style Guidance",
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600),
              ),
            ],
          ),
          const SizedBox(height: 6),
          const Text(
            "The AI uses these LaTeX templates as visual & structural blueprints (fonts, margins, macro structures) when tailoring for each job.",
            style: TextStyle(fontSize: 12, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _resumeTemplateController,
            decoration: const InputDecoration(
              labelText: "Resume Baseline Template Path",
              hintText: "templates/resume.tex",
              border: OutlineInputBorder(),
              helperText: "Path to baseline resume .tex inside your Overleaf project",
            ),
          ),
          const SizedBox(height: 14),
          TextField(
            controller: _coverLetterTemplateController,
            decoration: const InputDecoration(
              labelText: "Cover Letter Baseline Template Path",
              hintText: "templates/cover_letter.tex",
              border: OutlineInputBorder(),
              helperText: "Path to baseline cover letter .tex inside your Overleaf project",
            ),
          ),
          const SizedBox(height: 14),
          Row(
            children: [
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.open_in_browser, size: 16),
                  label: const Text("Edit in Overleaf"),
                  onPressed: () async {
                    final rawUrl = _overleafUrlController.text.trim();
                    if (rawUrl.isEmpty) {
                      ScaffoldMessenger.of(context).showSnackBar(
                        const SnackBar(content: Text("Please configure your Open-Overleaf URL first")),
                      );
                      return;
                    }
                    final project = _overleafProjectController.text.trim().isEmpty ? "job_applications" : _overleafProjectController.text.trim();
                    final uri = Uri.parse("${rawUrl.replaceAll(RegExp(r"/+$"), "")}/?project=$project");
                    if (await canLaunchUrl(uri)) {
                      await launchUrl(uri, mode: LaunchMode.externalApplication);
                    } else {
                      if (!mounted) return;
                      ScaffoldMessenger.of(context).showSnackBar(
                        SnackBar(content: Text("Could not launch: $uri")),
                      );
                    }
                  },
                ),
              ),
              const SizedBox(width: 10),
              Expanded(
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.refresh, size: 16),
                  label: const Text("Reset Defaults"),
                  onPressed: () async {
                    final ok = await ApiService().seedDefaultTemplates();
                    if (!mounted) return;
                    ScaffoldMessenger.of(context).showSnackBar(
                      SnackBar(
                        content: Text(
                          ok
                              ? "Default resume and cover letter formats restored in Open-Overleaf!"
                              : "Failed to restore default formats. Check server connection.",
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
              label: const Text("Save Overleaf & Format Settings"),
              onPressed: () async {
                final url = _overleafUrlController.text.trim();
                final secret = _overleafSecretController.text.trim();
                final project = _overleafProjectController.text.trim();
                final resumeTemplate = _resumeTemplateController.text.trim();
                final coverLetterTemplate = _coverLetterTemplateController.text.trim();

                if (url.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text("Please enter your Open-Overleaf Server URL")),
                  );
                  return;
                }

                if (!_hasConfiguredSecret && secret.isEmpty) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text("Please provide your MCP Secret / Access Token to secure your Open-Overleaf connection")),
                  );
                  return;
                }

                final ok = await ApiService().saveOverleafConfig(
                  deploymentUrl: url,
                  mcpSecret: secret.isNotEmpty ? secret : null,
                  projectName: project.isNotEmpty ? project : "job_applications",
                  resumeTemplatePath: resumeTemplate.isNotEmpty ? resumeTemplate : "templates/resume.tex",
                  coverLetterTemplatePath: coverLetterTemplate.isNotEmpty ? coverLetterTemplate : "templates/cover_letter.tex",
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
                          ? "Open-Overleaf configuration and format settings saved successfully!"
                          : "Failed to save Open-Overleaf configuration",
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
      title: "Target Page Limits",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            "Configure strict page budgets. The AI tailoring engine applies multi-pass tightening loops to fit your content cleanly across the full page without leaving empty white space.",
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          const Text(
            "Resume Target Length",
            style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: AppColors.primary),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [1, 2, 3, 4].map((page) {
              final isSelected = _targetResumePages == page;
              return ChoiceChip(
                label: Text("$page ${page == 1 ? "Page" : "Pages"}"),
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
            "Cover Letter Target Length",
            style: TextStyle(fontSize: 14, fontWeight: FontWeight.w600, color: AppColors.primary),
          ),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 8,
            children: [1, 2, 3, 4].map((page) {
              final isSelected = _targetCoverLetterPages == page;
              return ChoiceChip(
                label: Text("$page ${page == 1 ? "Page" : "Pages"}"),
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

  Widget _buildAppVersionCard() {
    final versionText = _appVersionDetails?.compactVersion ?? "v1.0.0+1";
    final platformText = _appVersionDetails?.platformName ?? (kIsWeb ? "Web" : "Mobile");
    final buildText = _appVersionDetails != null ? "Build ${_appVersionDetails!.buildNumber}" : "Build 1";

    return _buildSectionCard(
      icon: Icons.info_outline,
      title: "App Version & Environment",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            "Current client build specifications and deployment platform runtime.",
            style: TextStyle(fontSize: 13, color: AppColors.onSurfaceVariant),
          ),
          const SizedBox(height: 16),
          _buildVersionRow(
            icon: Icons.info_outline,
            label: "Version",
            value: versionText,
          ),
          const Divider(height: 20, color: AppColors.surfaceContainerHigh),
          _buildVersionRow(
            icon: kIsWeb ? Icons.language : Icons.devices,
            label: "Platform",
            value: platformText,
          ),
          const Divider(height: 20, color: AppColors.surfaceContainerHigh),
          _buildVersionRow(
            icon: Icons.tag,
            label: "Build Number",
            value: buildText,
          ),
        ],
      ),
    );
  }

  Widget _buildVersionRow({
    required IconData icon,
    required String label,
    required String value,
  }) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Row(
          children: [
            Icon(icon, color: AppColors.secondary, size: 20),
            const SizedBox(width: 8),
            Text(
              label,
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.w500,
                color: AppColors.onSurface,
              ),
            ),
          ],
        ),
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
          decoration: BoxDecoration(
            color: AppColors.surfaceContainerHigh,
            borderRadius: BorderRadius.circular(6),
          ),
          child: Text(
            value,
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w600,
              color: AppColors.onSurface,
            ),
          ),
        ),
      ],
    );
  }

  Widget _buildVersionFooter() {
    final displayText = _appVersionDetails?.displayVersion ?? "v1.0.0 • ${kIsWeb ? "Web" : "Mobile"}";
    return Center(
      child: Text(
        "Job Cruiser $displayText",
        style: const TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          color: AppColors.outline,
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
        "Preferences",
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
          tooltip: "Sign Out",
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
          "Job Search Preferences",
          style: TextStyle(
            fontSize: 28,
            fontWeight: FontWeight.bold,
            letterSpacing: -0.02,
            color: AppColors.primary,
            height: 1.2,
          ),
        ),
        const SizedBox(height: 4),
        const Text(
          "Fine-tune your criteria to receive higher-compatibility role matches and configure document tailoring settings.",
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
      title: "Target Industries",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                "ANY INDUSTRY / ALL INDUSTRIES",
                style: TextStyle(
                  fontFamily: "Geist",
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Switch(
                value: _anyIndustry,
                onChanged: (val) {
                  setState(() {
                    _anyIndustry = val;
                  });
                },
              ),
            ],
          ),
          if (!_anyIndustry) ...[
            const SizedBox(height: 12),
            const Text(
              "Select priority sectors (leave unchecked for all).",
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
        ],
      ),
    );
  }

  Widget _buildDesiredRoles() {
    return _buildSectionCard(
      icon: Icons.work,
      title: "Desired Roles",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                "ANY ROLE / ALL ROLES",
                style: TextStyle(
                  fontFamily: "Geist",
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Switch(
                value: _anyRole,
                onChanged: (val) {
                  setState(() {
                    _anyRole = val;
                  });
                },
              ),
            ],
          ),
          if (!_anyRole) ...[
            const SizedBox(height: 12),
            const Text(
              "QUICK ADD POPULAR ROLES",
              style: TextStyle(
                fontFamily: "Geist",
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
              "ADD CUSTOM ROLE",
              style: TextStyle(
                fontFamily: "Geist",
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
                  hintText: "e.g. Distributed Systems Engineer",
                  hintStyle: const TextStyle(
                    color: AppColors.outline,
                    fontSize: 14,
                  ),
                  contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                  border: InputBorder.none,
                  suffixIcon: IconButton(
                    icon: const Icon(Icons.add_circle, color: AppColors.primary),
                    onPressed: () {
                      final text = _roleController.text.trim();
                      if (text.isNotEmpty && !_currentTargets.contains(text)) {
                        setState(() {
                          _currentTargets.add(text);
                          _roleController.clear();
                        });
                      }
                    },
                  ),
                ),
                onSubmitted: (value) {
                  final text = value.trim();
                  if (text.isNotEmpty && !_currentTargets.contains(text)) {
                    setState(() {
                      _currentTargets.add(text);
                      _roleController.clear();
                    });
                  }
                },
              ),
            ),
            if (_currentTargets.isNotEmpty) ...[
              const SizedBox(height: 16),
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: _currentTargets.map((role) {
                  return Chip(
                    label: Text(role),
                    deleteIcon: const Icon(Icons.close, size: 16),
                    onDeleted: () {
                      setState(() {
                        _currentTargets.remove(role);
                      });
                    },
                    backgroundColor: AppColors.surfaceContainer,
                    side: const BorderSide(color: AppColors.outlineVariant),
                  );
                }).toList(),
              ),
            ],
          ],
        ],
      ),
    );
  }

  Widget _buildCompensationTarget() {
    final isINR = _currency == "INR";
    final minVal = 0.0;
    final maxVal = isINR ? 100.0 : 400.0;
    final currentVal = _baseSalary.clamp(minVal, maxVal);

    return _buildSectionCard(
      icon: Icons.attach_money,
      title: "Compensation Targets",
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                "ANY SALARY / NO MINIMUM",
                style: TextStyle(
                  fontFamily: "Geist",
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              Switch(
                value: _anySalary,
                onChanged: (val) {
                  setState(() {
                    _anySalary = val;
                    if (val) {
                      _baseSalary = 0.0;
                    }
                  });
                },
              ),
            ],
          ),
          if (!_anySalary) ...[
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  "CURRENCY",
                  style: TextStyle(
                    fontFamily: "Geist",
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    letterSpacing: 0.5,
                    color: AppColors.onSurfaceVariant,
                  ),
                ),
                Container(
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: AppColors.outlineVariant),
                  ),
                  child: SegmentedButton<String>(
                    segments: const [
                      ButtonSegment(value: "USD", label: Text("USD (\$)")),
                      ButtonSegment(value: "INR", label: Text("INR (₹)")),
                    ],
                    selected: {_currency},
                    onSelectionChanged: (newSelection) {
                      setState(() {
                        _currency = newSelection.first;
                        if (_currency == "INR" && _baseSalary > 100) {
                          _baseSalary = 0.0;
                        } else if (_currency == "USD" && _baseSalary > 400) {
                          _baseSalary = 0.0;
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
                  isINR ? "BASE SALARY (INR)" : "BASE SALARY (USD)",
                  style: const TextStyle(
                    fontFamily: "Geist",
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    letterSpacing: 0.5,
                    color: AppColors.onSurfaceVariant,
                  ),
                ),
                Text(
                  currentVal == 0.0
                      ? "Any / No Minimum"
                      : (isINR ? "₹${currentVal.toInt()} LPA+" : "\$${currentVal.toInt()}k+"),
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
                Text(isINR ? "₹0 LPA" : "\$0k", style: const TextStyle(fontSize: 11, color: AppColors.outline)),
                Text(isINR ? "₹100 LPA+" : "\$400k+", style: const TextStyle(fontSize: 11, color: AppColors.outline)),
              ],
            ),
          ],
          const SizedBox(height: 24),
          const Text(
            "EQUITY EXPECTATION",
            style: TextStyle(
              fontFamily: "Geist",
              fontSize: 12,
              fontWeight: FontWeight.w500,
              letterSpacing: 0.5,
              color: AppColors.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              _buildEquityOption("Standard"),
              const SizedBox(width: 8),
              _buildEquityOption("Meaningful"),
              const SizedBox(width: 8),
              _buildEquityOption("Founder Level"),
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
              Icon(icon, size: 20, color: AppColors.secondary),
              const SizedBox(width: 8),
              Text(
                title,
                style: const TextStyle(
                  fontFamily: "Geist",
                  fontSize: 14,
                  fontWeight: FontWeight.w600,
                  letterSpacing: -0.01,
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

  Widget _buildMatchNotificationCard() {
    return _buildSectionCard(
      icon: Icons.notifications_active_outlined,
      title: "Match Notifications",
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
                      "NOTIFY ON HIGH MATCHES",
                      style: TextStyle(
                        fontFamily: "Geist",
                        fontSize: 12,
                        fontWeight: FontWeight.w500,
                        color: AppColors.onSurfaceVariant,
                      ),
                    ),
                    SizedBox(height: 4),
                    Text(
                      "Receive in-app alerts and notifications when new job matches meet or exceed your percentage threshold.",
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
            const Divider(),
            const SizedBox(height: 12),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  "MATCH SCORE THRESHOLD",
                  style: TextStyle(
                    fontFamily: "Geist",
                    fontSize: 12,
                    fontWeight: FontWeight.w500,
                    letterSpacing: 0.5,
                    color: AppColors.onSurfaceVariant,
                  ),
                ),
                Text(
                  "${_matchThresholdPercentage.toInt()}%",
                  style: const TextStyle(
                    fontSize: 18,
                    fontWeight: FontWeight.bold,
                    color: AppColors.primary,
                  ),
                ),
              ],
            ),
            Slider(
              value: _matchThresholdPercentage.toDouble().clamp(50.0, 95.0),
              min: 50.0,
              max: 95.0,
              divisions: 9,
              label: "${_matchThresholdPercentage.toInt()}%",
              onChanged: (val) {
                setState(() {
                  _matchThresholdPercentage = val.round();
                });
              },
            ),
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: const [
                Text("50% (Broad)", style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
                Text("80% (Recommended)", style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
                Text("95% (Strict)", style: TextStyle(fontSize: 11, color: AppColors.onSurfaceVariant)),
              ],
            ),
            const SizedBox(height: 16),
            const Divider(),
            const SizedBox(height: 12),
            const Text(
              "CUSTOM NOTIFICATION CRITERIA (PROMPT)",
              style: TextStyle(
                fontFamily: "Geist",
                fontSize: 12,
                fontWeight: FontWeight.w500,
                letterSpacing: 0.5,
                color: AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 4),
            const Text(
              "Optional criteria prompt evaluated by AI matcher. Only jobs satisfying this prompt will trigger push alerts.",
              style: TextStyle(
                fontSize: 12,
                color: AppColors.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _notificationCriteriaController,
              maxLines: 2,
              style: const TextStyle(fontSize: 13),
              decoration: InputDecoration(
                hintText: "e.g. Only notify if 100% remote and mentions Kubernetes or Go",
                hintStyle: const TextStyle(fontSize: 12, color: AppColors.outlineVariant),
                filled: true,
                fillColor: AppColors.surfaceContainerLow,
                contentPadding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
                border: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.outlineVariant),
                ),
                enabledBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.outlineVariant),
                ),
                focusedBorder: OutlineInputBorder(
                  borderRadius: BorderRadius.circular(8),
                  borderSide: const BorderSide(color: AppColors.primary, width: 1.5),
                ),
              ),
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
          final apiService = ApiService();
          final profile = await apiService.fetchProfile();
          final String fetchedName = profile?["full_name"]?.toString() ?? "";
          final String fullName = fetchedName.trim().isNotEmpty && fetchedName != "User"
              ? fetchedName
              : "Dhruv";

          final int rawSalary = (_anySalary || _baseSalary <= 0)
              ? 0
              : (_currency == "INR"
                  ? _baseSalary.toInt() * 100000
                  : _baseSalary.toInt() * 1000);

          final targetRolesPayload = (_anyRole || _currentTargets.isEmpty) ? <String>["All Roles"] : _currentTargets.toList();
          final targetIndustriesPayload = (_anyIndustry || _selectedIndustries.isEmpty) ? <String>[] : _selectedIndustries.toList();
          final targetLocationsPayload = _anyLocation || _selectedLocations.isEmpty
              ? ["Any Location"]
              : _selectedLocations.toList();
          final workModelsPayload = _anyWorkModel || _selectedWorkModels.isEmpty
              ? ["any"]
              : _selectedWorkModels.toList();

          final saveSuccess = await apiService.savePreferences({
            "full_name": fullName,
            "target_roles": targetRolesPayload,
            "target_industries": targetIndustriesPayload,
            "target_locations": targetLocationsPayload,
            "work_models": workModelsPayload,
            "min_salary": rawSalary,
            "currency": _currency,
            "target_resume_pages": _targetResumePages,
            "target_cover_letter_pages": _targetCoverLetterPages,
            "match_threshold_notification_enabled": _matchThresholdNotificationEnabled,
            "match_threshold_percentage": _matchThresholdPercentage,
            "notification_prompt_criteria": _notificationCriteriaController.text.trim(),
          });

          if (!mounted) return;
          if (saveSuccess) {
            Navigator.pop(context, true);
          } else {
            ScaffoldMessenger.of(context).showSnackBar(
              const SnackBar(
                content: Text("Failed to save preferences. Please check server connection."),
                backgroundColor: AppColors.error,
              ),
            );
          }
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
              "Save Preferences",
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
