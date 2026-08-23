import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
import 'package:syncfusion_flutter_pdf/pdf.dart';
import 'auth.dart';
import 'main.dart' show AppColors, JobCruiserShell;
import 'services/api_service.dart';

/// Multi-step Onboarding Wizard screen for new users.
class OnboardingWizardScreen extends StatefulWidget {
  const OnboardingWizardScreen({super.key, this.suggestedName});

  final String? suggestedName;

  @override
  State<OnboardingWizardScreen> createState() => _OnboardingWizardScreenState();
}

class _OnboardingWizardScreenState extends State<OnboardingWizardScreen> {
  final ApiService _apiService = ApiService();
  int _currentStep = 0;
  bool _isSaving = false;

  late TextEditingController _nameController;
  late TextEditingController _bioTextController;
  late TextEditingController _overleafUrlController;
  late TextEditingController _overleafSecretController;
  late TextEditingController _overleafProjectController;
  late TextEditingController _locationController;

  bool _anyRole = true;
  bool _anyIndustry = true;
  bool _anySalary = true;
  bool _anyLocation = false;
  bool _anyWorkModel = false;
  bool _isParsingCV = false;
  String _rawCvText = '';

  final Set<String> _selectedLocations = {'India (On-site & Hybrid)', 'India (Remote)', 'Global Remote'};
  final List<String> _availableLocations = [
    'India (On-site & Hybrid)',
    'India (Remote)',
    'Global Remote',
    'US / North America Remote',
    'Europe Remote',
  ];

  final List<Map<String, String>> _experiences = [];
  final List<Map<String, String>> _projects = [];
  final List<Map<String, String>> _education = [];
  final List<String> _skills = [];
  final List<Map<String, String>> _achievements = [];
  final List<Map<String, String>> _certifications = [];

  double _minSalary = 120.0;
  final Set<String> _selectedRoles = {'Backend Engineer', 'Fullstack SDE'};
  final Set<String> _selectedIndustries = {'Fintech', 'AI / ML', 'Enterprise SaaS'};
  final Set<String> _selectedWorkModels = {'remote', 'hybrid'};
  final List<String> _availableWorkModels = ['remote', 'hybrid', 'onsite'];

  final List<String> _availableRoles = [
    'Backend Engineer',
    'Fullstack SDE',
    'Frontend Engineer',
    'DevOps SRE',
    'Cloud Platform Engineer',
    'Systems Software Engineer',
    'Rust Engineer',
    'AI / ML Engineer',
    'Data Platform Engineer',
    'Embedded / Low-Level Engineer',
    'QA / Automation Engineer',
    'Security / Infrastructure Engineer',
    'Product Manager',
  ];

  final List<String> _availableIndustries = [
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

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.suggestedName ?? '');
    _locationController = TextEditingController();
    _bioTextController = TextEditingController();
    _overleafUrlController = TextEditingController();
    _overleafSecretController = TextEditingController();
    _overleafProjectController = TextEditingController(text: 'job_applications');
  }

  @override
  void dispose() {
    _nameController.dispose();
    _locationController.dispose();
    _bioTextController.dispose();
    _overleafUrlController.dispose();
    _overleafSecretController.dispose();
    _overleafProjectController.dispose();
    super.dispose();
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
        _rawCvText = extractedText.trim();
      });

      final parsed = await _apiService.parseCVWithGemini(extractedText.trim());

      if (!mounted) return;
      setState(() {
        _isParsingCV = false;
        if (parsed != null) {
          if (parsed['bio_summary'] != null && (parsed['bio_summary'] as String).isNotEmpty) {
            _bioTextController.text = parsed['bio_summary'] as String;
          } else {
            _bioTextController.text = extractedText.trim();
          }

          if (parsed['experience'] is List) {
            _experiences.clear();
            for (var item in parsed['experience']) {
              _experiences.add({
                'company': item['company']?.toString() ?? '',
                'role': item['role']?.toString() ?? '',
                'duration': item['duration']?.toString() ?? '',
                'highlights': item['highlights']?.toString() ?? '',
              });
            }
          }

          if (parsed['projects'] is List) {
            _projects.clear();
            for (var item in parsed['projects']) {
              final ts = (item['tech_stack'] is List)
                  ? (item['tech_stack'] as List).join(', ')
                  : (item['tech_stack']?.toString() ?? '');
              _projects.add({
                'title': item['title']?.toString() ?? '',
                'tech_stack': ts,
                'description': item['description']?.toString() ?? '',
                'link': item['link']?.toString() ?? '',
              });
            }
          }

          if (parsed['achievements'] is List) {
            _achievements.clear();
            for (var item in parsed['achievements']) {
              _achievements.add({
                'title': item['title']?.toString() ?? '',
                'details': item['details']?.toString() ?? '',
              });
            }
          }

          if (parsed['location'] != null && (parsed['location'] as String).isNotEmpty) {
            _locationController.text = parsed['location'] as String;
          }

          if (parsed['skills'] is List) {
            _skills.clear();
            for (var sk in parsed['skills']) {
              if (sk != null && sk.toString().isNotEmpty) {
                _skills.add(sk.toString());
              }
            }
          }

          if (parsed['education'] is List) {
            _education.clear();
            for (var item in parsed['education']) {
              _education.add({
                'institution': item['institution']?.toString() ?? '',
                'degree': item['degree']?.toString() ?? '',
                'year': item['year']?.toString() ?? '',
                'grade': item['grade']?.toString() ?? '',
              });
            }
          }
        } else {
          _bioTextController.text = extractedText.trim();
        }
      });

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            parsed != null
                ? 'Parsed resume into structured Experience, Projects & Skills!'
                : 'Extracted resume text from ${file.name}!',
          ),
        ),
      );
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _isParsingCV = false;
      });
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Error picking/parsing file: $e')),
      );
    }
  }

  Future<void> _completeOnboarding() async {
    setState(() {
      _isSaving = true;
    });

    final targetRoles = _anyRole ? ['Any Role'] : _selectedRoles.toList();
    final targetIndustries = _anyIndustry ? ['Any Industry'] : _selectedIndustries.toList();
    final targetLocations = _anyLocation ? ['Any Location'] : _selectedLocations.toList();
    final workModels = _anyWorkModel ? ['any'] : _selectedWorkModels.toList();
    final minSalaryVal = _anySalary ? 0 : (_minSalary.toInt() * 1000);

    Map<String, dynamic> fullCvPayload = {
      'raw_cv': _rawCvText,
      'experiences': _experiences,
      'projects': _projects,
      'education': _education,
      'skills': _skills,
      'achievements': _achievements,
      'certifications': _certifications,
    };
    String masterCvString = jsonEncode(fullCvPayload);
    if (_rawCvText.isNotEmpty) {
      masterCvString = "$_rawCvText\n\n--- STRUCTURED RESUME DETAILS ---\n$masterCvString";
    }

    final prefSuccess = await _apiService.savePreferences({
      'full_name': _nameController.text.trim().isNotEmpty
          ? _nameController.text.trim()
          : 'User',
      'target_roles': targetRoles,
      'target_industries': targetIndustries,
      'target_locations': targetLocations,
      'work_models': workModels,
      'min_salary': minSalaryVal,
      'currency': 'USD',
      'bio_experience_text': _bioTextController.text.trim(),
      'master_cv_text': masterCvString,
      'target_resume_pages': 1,
      'target_cover_letter_pages': 1,
    });

    if (_overleafUrlController.text.trim().isNotEmpty) {
      final secret = _overleafSecretController.text.trim();
      final project = _overleafProjectController.text.trim();
      await _apiService.saveOverleafConfig(
        deploymentUrl: _overleafUrlController.text.trim(),
        mcpSecret: secret.isNotEmpty ? secret : null,
        projectName: project.isNotEmpty ? project : 'job_applications',
      );
    }

    if (!mounted) return;

    setState(() {
      _isSaving = false;
    });

    if (prefSuccess) {
      Navigator.of(context).pushReplacement(
        MaterialPageRoute(builder: (_) => const JobCruiserShell()),
      );
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Failed to save profile preferences.')),
      );
    }
  }

  Future<void> _handleSignOut() async {
    await _apiService.clearToken();
    if (!mounted) return;
    Navigator.pushAndRemoveUntil(
      context,
      MaterialPageRoute(builder: (_) => const AuthScreen()),
      (route) => false,
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: AppColors.surface,
        elevation: 0,
        title: const Text(
          'Account Setup & Preferences',
          style: TextStyle(
            color: AppColors.primary,
            fontSize: 20,
            fontWeight: FontWeight.bold,
          ),
        ),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout, color: AppColors.error),
            tooltip: 'Sign Out',
            onPressed: _handleSignOut,
          ),
        ],
      ),
      body: _isSaving
          ? const Center(child: CircularProgressIndicator())
          : Stepper(
              currentStep: _currentStep,
              onStepContinue: () {
                if (_currentStep < 2) {
                  setState(() {
                    _currentStep++;
                  });
                } else {
                  _completeOnboarding();
                }
              },
              onStepCancel: () {
                if (_currentStep > 0) {
                  setState(() {
                    _currentStep--;
                  });
                }
              },
              controlsBuilder: (context, details) {
                return Padding(
                  padding: const EdgeInsets.only(top: 24),
                  child: Row(
                    children: [
                      ElevatedButton(
                        onPressed: details.onStepContinue,
                        style: ElevatedButton.styleFrom(
                          backgroundColor: AppColors.primary,
                          foregroundColor: Colors.white,
                          padding: const EdgeInsets.symmetric(
                            horizontal: 24,
                            vertical: 12,
                          ),
                        ),
                        child: Text(
                          _currentStep == 2 ? 'Complete & Start' : 'Next Step',
                        ),
                      ),
                      if (_currentStep > 0) ...[
                        const SizedBox(width: 12),
                        OutlinedButton(
                          onPressed: details.onStepCancel,
                          style: OutlinedButton.styleFrom(
                            foregroundColor: AppColors.primary,
                            padding: const EdgeInsets.symmetric(
                              horizontal: 24,
                              vertical: 12,
                            ),
                          ),
                          child: const Text('Back'),
                        ),
                      ],
                    ],
                  ),
                );
              },
              steps: [
                Step(
                  title: const Text('Basic Profile & Self-Hosted open-overleaf'),
                  isActive: _currentStep >= 0,
                  content: _buildStep1(),
                ),
                Step(
                  title: const Text('Experience & Resume Parser'),
                  isActive: _currentStep >= 1,
                  content: _buildStep2(),
                ),
                Step(
                  title: const Text('Roles, Salary & Work Preferences'),
                  isActive: _currentStep >= 2,
                  content: _buildStep3(),
                ),
              ],
            ),
    );
  }

  Widget _buildStep1() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Let\'s get your profile set up',
          style: TextStyle(
            fontSize: 16,
            fontWeight: FontWeight.bold,
            color: AppColors.primary,
          ),
        ),
        const SizedBox(height: 8),
        const Text(
          'Enter your full name and basic preferences to get started with job recommendations.',
          style: TextStyle(fontSize: 14, color: AppColors.onSurfaceVariant),
        ),
        const SizedBox(height: 24),
        TextField(
          controller: _nameController,
          decoration: const InputDecoration(
            labelText: 'Full Name',
            hintText: 'e.g. Jane Doe',
            border: OutlineInputBorder(),
            contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          ),
        ),
        const SizedBox(height: 24),
        const Text(
          'Self-Hosted Open-Overleaf Integration (Optional)',
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.bold,
            color: AppColors.primary,
          ),
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _overleafUrlController,
          decoration: const InputDecoration(
            labelText: 'Server Base URL',
            hintText: 'e.g. https://overleaf.example.com',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _overleafSecretController,
          obscureText: true,
          decoration: const InputDecoration(
            labelText: 'MCP Secret / Access Token',
            hintText: 'OVERLEAF_MCP_SECRET or OVERLEAF_MCP_TOKEN',
            helperText: 'Required if connecting Open-Overleaf. Encrypted at rest (AES-256).',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _overleafProjectController,
          decoration: const InputDecoration(
            labelText: 'Project Name (Optional)',
            hintText: 'job_applications',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
      ],
    );
  }

  Widget _buildStep2() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            Expanded(
              child: OutlinedButton.icon(
                onPressed: _isParsingCV ? null : _extractPdfToBio,
                icon: _isParsingCV
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.picture_as_pdf, color: AppColors.primary),
                label: Text(
                  _isParsingCV ? 'Parsing with Gemini AI...' : 'Import & AI-Parse CV',
                ),
                style: OutlinedButton.styleFrom(
                  foregroundColor: AppColors.primary,
                  side: const BorderSide(color: AppColors.outlineVariant),
                  padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 16),
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 20),
        DefaultTabController(
          length: 7,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const TabBar(
                isScrollable: true,
                labelColor: AppColors.primary,
                indicatorColor: AppColors.primary,
                tabs: [
                  Tab(text: 'Experience'),
                  Tab(text: 'Projects'),
                  Tab(text: 'Education'),
                  Tab(text: 'Skills & Tech'),
                  Tab(text: 'Achievements'),
                  Tab(text: 'Certifications'),
                  Tab(text: 'Bio & Location'),
                ],
              ),
              const SizedBox(height: 16),
              SizedBox(
                height: 420,
                child: TabBarView(
                  children: [
                    _buildExperienceTab(),
                    _buildProjectsTab(),
                    _buildEducationTab(),
                    _buildSkillsTab(),
                    _buildAchievementsTab(),
                    _buildCertificationsTab(),
                    _buildBioSummaryTab(),
                  ],
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  Widget _buildExperienceTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Work Experience',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Experience',
              onPressed: () => _showAddExperienceDialog(),
            ),
          ],
        ),
        Expanded(
          child: _experiences.isEmpty
              ? const Center(
                  child: Text('No work experience added yet. Upload CV or tap + to add.'),
                )
              : ListView.builder(
                  itemCount: _experiences.length,
                  itemBuilder: (context, index) {
                    final item = _experiences[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        title: Text(
                          '${item['role']} @ ${item['company']}',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text(
                          '${item['duration']}\n${item['highlights']}',
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined, color: AppColors.primary),
                              onPressed: () => _showAddExperienceDialog(editIndex: index),
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppColors.error),
                              onPressed: () {
                                setState(() {
                                  _experiences.removeAt(index);
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildProjectsTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Key Projects',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Project',
              onPressed: () => _showAddProjectDialog(),
            ),
          ],
        ),
        Expanded(
          child: _projects.isEmpty
              ? const Center(
                  child: Text('No projects added yet. Upload CV or tap + to add.'),
                )
              : ListView.builder(
                  itemCount: _projects.length,
                  itemBuilder: (context, index) {
                    final item = _projects[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        title: Text(
                          item['title'] ?? '',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text(
                          'Tech: ${item['tech_stack']}\n${item['description']}',
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined, color: AppColors.primary),
                              onPressed: () => _showAddProjectDialog(editIndex: index),
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppColors.error),
                              onPressed: () {
                                setState(() {
                                  _projects.removeAt(index);
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildEducationTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Education History',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Education',
              onPressed: () => _showAddEducationDialog(),
            ),
          ],
        ),
        Expanded(
          child: _education.isEmpty
              ? const Center(
                  child: Text('No education history added yet. Upload CV or tap + to add.'),
                )
              : ListView.builder(
                  itemCount: _education.length,
                  itemBuilder: (context, index) {
                    final item = _education[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        title: Text(
                          '${item['degree']} - ${item['institution']}',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text('Dates: ${item['year']} | Grade: ${item['grade']}'),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined, color: AppColors.primary),
                              onPressed: () => _showAddEducationDialog(editIndex: index),
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppColors.error),
                              onPressed: () {
                                setState(() {
                                  _education.removeAt(index);
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildSkillsTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Skills & Technical Proficiencies',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Skill',
              onPressed: _showAddSkillDialog,
            ),
          ],
        ),
        const SizedBox(height: 8),
        Expanded(
          child: _skills.isEmpty
              ? const Center(
                  child: Text('No skills added yet. Upload CV or tap + to add.'),
                )
              : SingleChildScrollView(
                  child: Wrap(
                    spacing: 8,
                    runSpacing: 8,
                    children: _skills.map((sk) {
                      return Chip(
                        label: Text(sk),
                        onDeleted: () {
                          setState(() {
                            _skills.remove(sk);
                          });
                        },
                      );
                    }).toList(),
                  ),
                ),
        ),
      ],
    );
  }

  Widget _buildAchievementsTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Achievements & Awards',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Achievement',
              onPressed: () => _showAddAchievementDialog(),
            ),
          ],
        ),
        Expanded(
          child: _achievements.isEmpty
              ? const Center(
                  child: Text('No achievements added yet. Upload CV or tap + to add.'),
                )
              : ListView.builder(
                  itemCount: _achievements.length,
                  itemBuilder: (context, index) {
                    final item = _achievements[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        title: Text(
                          item['title'] ?? '',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text(item['details'] ?? ''),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined, color: AppColors.primary),
                              onPressed: () => _showAddAchievementDialog(editIndex: index),
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppColors.error),
                              onPressed: () {
                                setState(() {
                                  _achievements.removeAt(index);
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildCertificationsTab() {
    return Column(
      children: [
        Row(
          children: [
            const Expanded(
              child: Text(
                'Certifications & Credentials',
                style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary, fontSize: 16),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            IconButton(
              icon: const Icon(Icons.add_circle, color: AppColors.primary),
              tooltip: 'Add Certification',
              onPressed: () => _showAddCertificationDialog(),
            ),
          ],
        ),
        Expanded(
          child: _certifications.isEmpty
              ? const Center(
                  child: Text('No certifications added yet. Upload CV or tap + to add.'),
                )
              : ListView.builder(
                  itemCount: _certifications.length,
                  itemBuilder: (context, index) {
                    final item = _certifications[index];
                    return Card(
                      margin: const EdgeInsets.only(bottom: 8),
                      shape: RoundedRectangleBorder(
                        borderRadius: BorderRadius.circular(8),
                        side: const BorderSide(color: AppColors.outlineVariant),
                      ),
                      child: ListTile(
                        title: Text(
                          item['name'] ?? '',
                          style: const TextStyle(fontWeight: FontWeight.bold),
                        ),
                        subtitle: Text('Issuer: ${item['issuer']}'),
                        trailing: Row(
                          mainAxisSize: MainAxisSize.min,
                          children: [
                            IconButton(
                              icon: const Icon(Icons.edit_outlined, color: AppColors.primary),
                              onPressed: () => _showAddCertificationDialog(editIndex: index),
                            ),
                            IconButton(
                              icon: const Icon(Icons.delete_outline, color: AppColors.error),
                              onPressed: () {
                                setState(() {
                                  _certifications.removeAt(index);
                                });
                              },
                            ),
                          ],
                        ),
                      ),
                    );
                  },
                ),
        ),
      ],
    );
  }

  Widget _buildBioSummaryTab() {
    return SingleChildScrollView(
      child: Padding(
        padding: const EdgeInsets.only(bottom: 12.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            TextField(
              controller: _locationController,
              decoration: const InputDecoration(
                labelText: 'Primary Location (City, Country)',
                hintText: 'e.g. San Francisco, USA or Remote',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 16),
            const Text(
              'First-Person Bio & Summary',
              style: TextStyle(fontWeight: FontWeight.bold, color: AppColors.primary),
            ),
            const SizedBox(height: 8),
            SizedBox(
              height: 180,
              child: TextField(
                controller: _bioTextController,
                maxLines: 8,
                decoration: const InputDecoration(
                  hintText: 'Paste or edit your first-person summary here (e.g. "I am a Full Stack Engineer...")...',
                  border: OutlineInputBorder(),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }

  Future<String?> _showMonthYearPicker({
    required BuildContext context,
    required String title,
    String? initialMonthYear,
    bool allowPresent = false,
  }) async {
    final months = [
      'Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun',
      'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'
    ];
    final currentYear = DateTime.now().year;
    final years = List<int>.generate(35, (i) => currentYear - 25 + i);

    String selectedMonth = 'Jan';
    int selectedYear = currentYear;
    bool isPresent = false;

    if (initialMonthYear != null && initialMonthYear.trim().isNotEmpty) {
      if (initialMonthYear.trim().toLowerCase() == 'present') {
        isPresent = true;
      } else {
        final parts = initialMonthYear.trim().split(' ');
        if (parts.length >= 2) {
          if (months.contains(parts[0])) selectedMonth = parts[0];
          final y = int.tryParse(parts[1]);
          if (y != null) selectedYear = y;
        }
      }
    }

    return showDialog<String>(
      context: context,
      builder: (ctx) {
        return StatefulBuilder(
          builder: (ctx, setDlgState) {
            return AlertDialog(
              title: Text(title),
              content: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (allowPresent) ...[
                    CheckboxListTile(
                      title: const Text('Currently Work / Enrolled Here (Present)'),
                      value: isPresent,
                      onChanged: (val) {
                        setDlgState(() {
                          isPresent = val ?? false;
                        });
                      },
                      controlAffinity: ListTileControlAffinity.leading,
                      contentPadding: EdgeInsets.zero,
                    ),
                    const Divider(),
                  ],
                  if (!isPresent) ...[
                    Row(
                      children: [
                        Expanded(
                          child: DropdownButtonFormField<String>(
                            initialValue: selectedMonth,
                            decoration: const InputDecoration(labelText: 'Month', border: OutlineInputBorder()),
                            items: months.map((m) => DropdownMenuItem(value: m, child: Text(m))).toList(),
                            onChanged: (val) {
                              if (val != null) {
                                setDlgState(() {
                                  selectedMonth = val;
                                });
                              }
                            },
                          ),
                        ),
                        const SizedBox(width: 8),
                        Expanded(
                          child: DropdownButtonFormField<int>(
                            initialValue: selectedYear,
                            decoration: const InputDecoration(labelText: 'Year', border: OutlineInputBorder()),
                            items: years.map((y) => DropdownMenuItem(value: y, child: Text('$y'))).toList(),
                            onChanged: (val) {
                              if (val != null) {
                                setDlgState(() {
                                  selectedYear = val;
                                });
                              }
                            },
                          ),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
              actions: [
                TextButton(
                  onPressed: () => Navigator.pop(ctx),
                  child: const Text('Cancel'),
                ),
                ElevatedButton(
                  onPressed: () {
                    if (isPresent) {
                      Navigator.pop(ctx, 'Present');
                    } else {
                      Navigator.pop(ctx, '$selectedMonth $selectedYear');
                    }
                  },
                  child: const Text('Select'),
                ),
              ],
            );
          },
        );
      },
    );
  }

  void _showAddExperienceDialog({int? editIndex}) {
    final isEditing = editIndex != null;
    final companyCtrl = TextEditingController(text: isEditing ? _experiences[editIndex]['company'] : '');
    final roleCtrl = TextEditingController(text: isEditing ? _experiences[editIndex]['role'] : '');
    final durationCtrl = TextEditingController(text: isEditing ? _experiences[editIndex]['duration'] : '');
    final highlightsCtrl = TextEditingController(text: isEditing ? _experiences[editIndex]['highlights'] : '');

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDlgState) {
          return AlertDialog(
            title: Text(isEditing ? 'Edit Work Experience' : 'Add Work Experience'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(controller: companyCtrl, decoration: const InputDecoration(labelText: 'Company')),
                  const SizedBox(height: 8),
                  TextField(controller: roleCtrl, decoration: const InputDecoration(labelText: 'Role / Title')),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () async {
                            final res = await _showMonthYearPicker(
                              context: context,
                              title: 'Select Start Month & Year',
                            );
                            if (res != null) {
                              setDlgState(() {
                                final parts = durationCtrl.text.split(' - ');
                                final endPart = parts.length > 1 ? parts[1] : 'Present';
                                durationCtrl.text = '$res - $endPart';
                              });
                            }
                          },
                          icon: const Icon(Icons.calendar_month, size: 16),
                          label: const Text('Start Date'),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () async {
                            final res = await _showMonthYearPicker(
                              context: context,
                              title: 'Select End Month & Year',
                              allowPresent: true,
                            );
                            if (res != null) {
                              setDlgState(() {
                                final parts = durationCtrl.text.split(' - ');
                                final startPart = parts.isNotEmpty && parts[0].trim().isNotEmpty ? parts[0] : 'Jan 2023';
                                durationCtrl.text = '$startPart - $res';
                              });
                            }
                          },
                          icon: const Icon(Icons.event_available, size: 16),
                          label: const Text('End Date'),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: durationCtrl,
                    decoration: const InputDecoration(
                      labelText: 'Duration Range',
                      hintText: 'e.g. Nov 2021 - Present',
                      isDense: true,
                    ),
                  ),
                  const SizedBox(height: 8),
                  TextField(
                    controller: highlightsCtrl,
                    maxLines: 3,
                    decoration: const InputDecoration(labelText: 'Highlights & Responsibilities'),
                  ),
                ],
              ),
            ),
            actions: [
              TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
              ElevatedButton(
                onPressed: () {
                  if (roleCtrl.text.trim().isNotEmpty) {
                    setState(() {
                      final data = {
                        'company': companyCtrl.text.trim(),
                        'role': roleCtrl.text.trim(),
                        'duration': durationCtrl.text.trim(),
                        'highlights': highlightsCtrl.text.trim(),
                      };
                      if (isEditing) {
                        _experiences[editIndex] = data;
                      } else {
                        _experiences.add(data);
                      }
                    });
                  }
                  Navigator.pop(ctx);
                },
                child: Text(isEditing ? 'Save' : 'Add'),
              ),
            ],
          );
        },
      ),
    );
  }

  void _showAddProjectDialog({int? editIndex}) {
    final isEditing = editIndex != null;
    final titleCtrl = TextEditingController(text: isEditing ? _projects[editIndex]['title'] : '');
    final techCtrl = TextEditingController(text: isEditing ? _projects[editIndex]['tech_stack'] : '');
    final descCtrl = TextEditingController(text: isEditing ? _projects[editIndex]['description'] : '');
    final linkCtrl = TextEditingController(text: isEditing ? _projects[editIndex]['link'] : '');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(isEditing ? 'Edit Key Project' : 'Add Key Project'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: titleCtrl, decoration: const InputDecoration(labelText: 'Project Title')),
              TextField(controller: techCtrl, decoration: const InputDecoration(labelText: 'Tech Stack (e.g. Go, React)')),
              TextField(controller: descCtrl, maxLines: 3, decoration: const InputDecoration(labelText: 'Description')),
              TextField(controller: linkCtrl, decoration: const InputDecoration(labelText: 'Project Link / URL')),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              if (titleCtrl.text.trim().isNotEmpty) {
                setState(() {
                  final data = {
                    'title': titleCtrl.text.trim(),
                    'tech_stack': techCtrl.text.trim(),
                    'description': descCtrl.text.trim(),
                    'link': linkCtrl.text.trim(),
                  };
                  if (isEditing) {
                    _projects[editIndex] = data;
                  } else {
                    _projects.add(data);
                  }
                });
              }
              Navigator.pop(ctx);
            },
            child: Text(isEditing ? 'Save' : 'Add'),
          ),
        ],
      ),
    );
  }

  void _showAddEducationDialog({int? editIndex}) {
    final isEditing = editIndex != null;
    final instCtrl = TextEditingController(text: isEditing ? _education[editIndex]['institution'] : '');
    final degreeCtrl = TextEditingController(text: isEditing ? _education[editIndex]['degree'] : '');
    final yearCtrl = TextEditingController(text: isEditing ? _education[editIndex]['year'] : '');
    final gradeCtrl = TextEditingController(text: isEditing ? _education[editIndex]['grade'] : '');

    showDialog(
      context: context,
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setDlgState) {
          return AlertDialog(
            title: Text(isEditing ? 'Edit Education' : 'Add Education'),
            content: SingleChildScrollView(
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  TextField(controller: instCtrl, decoration: const InputDecoration(labelText: 'School / Institution')),
                  const SizedBox(height: 8),
                  TextField(controller: degreeCtrl, decoration: const InputDecoration(labelText: 'Degree / Program')),
                  const SizedBox(height: 12),
                  Row(
                    children: [
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () async {
                            final res = await _showMonthYearPicker(
                              context: context,
                              title: 'Select Start Month & Year',
                            );
                            if (res != null) {
                              setDlgState(() {
                                final parts = yearCtrl.text.split(' - ');
                                final endPart = parts.length > 1 ? parts[1] : 'Present';
                                yearCtrl.text = '$res - $endPart';
                              });
                            }
                          },
                          icon: const Icon(Icons.calendar_month, size: 16),
                          label: const Text('Start Date'),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: OutlinedButton.icon(
                          onPressed: () async {
                            final res = await _showMonthYearPicker(
                              context: context,
                              title: 'Select End Month & Year',
                              allowPresent: true,
                            );
                            if (res != null) {
                              setDlgState(() {
                                final parts = yearCtrl.text.split(' - ');
                                final startPart = parts.isNotEmpty && parts[0].trim().isNotEmpty ? parts[0] : 'Apr 2023';
                                yearCtrl.text = '$startPart - $res';
                              });
                            }
                          },
                          icon: const Icon(Icons.event_available, size: 16),
                          label: const Text('End Date'),
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  TextField(controller: yearCtrl, decoration: const InputDecoration(labelText: 'Dates / Year Range')),
                  const SizedBox(height: 8),
                  TextField(controller: gradeCtrl, decoration: const InputDecoration(labelText: 'Grade / CGPA')),
                ],
              ),
            ),
            actions: [
              TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
              ElevatedButton(
                onPressed: () {
                  if (instCtrl.text.trim().isNotEmpty || degreeCtrl.text.trim().isNotEmpty) {
                    setState(() {
                      final data = {
                        'institution': instCtrl.text.trim(),
                        'degree': degreeCtrl.text.trim(),
                        'year': yearCtrl.text.trim(),
                        'grade': gradeCtrl.text.trim(),
                      };
                      if (isEditing) {
                        _education[editIndex] = data;
                      } else {
                        _education.add(data);
                      }
                    });
                  }
                  Navigator.pop(ctx);
                },
                child: Text(isEditing ? 'Save' : 'Add'),
              ),
            ],
          );
        },
      ),
    );
  }

  void _showAddSkillDialog() {
    final skillCtrl = TextEditingController();

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Add Technical Skill'),
        content: TextField(
          controller: skillCtrl,
          decoration: const InputDecoration(labelText: 'Skill / Tech Stack (e.g. Docker, Rust)'),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              final val = skillCtrl.text.trim();
              if (val.isNotEmpty && !_skills.contains(val)) {
                setState(() {
                  _skills.add(val);
                });
              }
              Navigator.pop(ctx);
            },
            child: const Text('Add'),
          ),
        ],
      ),
    );
  }

  void _showAddAchievementDialog({int? editIndex}) {
    final isEditing = editIndex != null;
    final titleCtrl = TextEditingController(text: isEditing ? _achievements[editIndex]['title'] : '');
    final detailsCtrl = TextEditingController(text: isEditing ? _achievements[editIndex]['details'] : '');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(isEditing ? 'Edit Achievement' : 'Add Achievement'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: titleCtrl, decoration: const InputDecoration(labelText: 'Achievement Title')),
              TextField(controller: detailsCtrl, maxLines: 3, decoration: const InputDecoration(labelText: 'Details')),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              if (titleCtrl.text.trim().isNotEmpty) {
                setState(() {
                  final data = {
                    'title': titleCtrl.text.trim(),
                    'details': detailsCtrl.text.trim(),
                  };
                  if (isEditing) {
                    _achievements[editIndex] = data;
                  } else {
                    _achievements.add(data);
                  }
                });
              }
              Navigator.pop(ctx);
            },
            child: Text(isEditing ? 'Save' : 'Add'),
          ),
        ],
      ),
    );
  }

  void _showAddCertificationDialog({int? editIndex}) {
    final isEditing = editIndex != null;
    final nameCtrl = TextEditingController(text: isEditing ? _certifications[editIndex]['name'] : '');
    final issuerCtrl = TextEditingController(text: isEditing ? _certifications[editIndex]['issuer'] : '');

    showDialog(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(isEditing ? 'Edit Certification' : 'Add Certification'),
        content: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(controller: nameCtrl, decoration: const InputDecoration(labelText: 'Certification Name')),
              TextField(controller: issuerCtrl, decoration: const InputDecoration(labelText: 'Issuing Organization')),
            ],
          ),
        ),
        actions: [
          TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          ElevatedButton(
            onPressed: () {
              if (nameCtrl.text.trim().isNotEmpty) {
                setState(() {
                  final data = {
                    'name': nameCtrl.text.trim(),
                    'issuer': issuerCtrl.text.trim(),
                  };
                  if (isEditing) {
                    _certifications[editIndex] = data;
                  } else {
                    _certifications.add(data);
                  }
                });
              }
              Navigator.pop(ctx);
            },
            child: Text(isEditing ? 'Save' : 'Add'),
          ),
        ],
      ),
    );
  }

  Widget _buildStep3() {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Target Roles',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: AppColors.primary,
              ),
            ),
            FilterChip(
              label: const Text('Any Role'),
              selected: _anyRole,
              onSelected: (val) {
                setState(() {
                  _anyRole = val;
                });
              },
            ),
          ],
        ),
        if (!_anyRole) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: _availableRoles.map((role) {
              final isSelected = _selectedRoles.contains(role);
              return FilterChip(
                label: Text(role),
                selected: isSelected,
                onSelected: (val) {
                  setState(() {
                    if (val) {
                      _selectedRoles.add(role);
                    } else {
                      _selectedRoles.remove(role);
                    }
                  });
                },
              );
            }).toList(),
          ),
        ],
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Preferred Industries',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: AppColors.primary,
              ),
            ),
            FilterChip(
              label: const Text('Any Industry'),
              selected: _anyIndustry,
              onSelected: (val) {
                setState(() {
                  _anyIndustry = val;
                });
              },
            ),
          ],
        ),
        if (!_anyIndustry) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: _availableIndustries.map((ind) {
              final isSelected = _selectedIndustries.contains(ind);
              return FilterChip(
                label: Text(ind),
                selected: isSelected,
                onSelected: (val) {
                  setState(() {
                    if (val) {
                      _selectedIndustries.add(ind);
                    } else {
                      _selectedIndustries.remove(ind);
                    }
                  });
                },
              );
            }).toList(),
          ),
        ],
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Target Job Locations',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: AppColors.primary,
              ),
            ),
            FilterChip(
              label: const Text('Any Location'),
              selected: _anyLocation,
              onSelected: (val) {
                setState(() {
                  _anyLocation = val;
                });
              },
            ),
          ],
        ),
        if (!_anyLocation) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: _availableLocations.map((loc) {
              final isSelected = _selectedLocations.contains(loc);
              return FilterChip(
                label: Text(loc),
                selected: isSelected,
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
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Text(
              _anySalary
                  ? 'Min Salary: Any Salary'
                  : 'Min Salary: \$${_minSalary.toInt()}k+',
              style: const TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: AppColors.primary,
              ),
            ),
            FilterChip(
              label: const Text('Any Salary'),
              selected: _anySalary,
              onSelected: (val) {
                setState(() {
                  _anySalary = val;
                });
              },
            ),
          ],
        ),
        if (!_anySalary)
          Slider(
            value: _minSalary,
            min: 50,
            max: 300,
            divisions: 25,
            label: '\$${_minSalary.toInt()}k+',
            onChanged: (val) {
              setState(() {
                _minSalary = val;
              });
            },
          ),
        const SizedBox(height: 16),
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Work Model',
              style: TextStyle(
                fontSize: 14,
                fontWeight: FontWeight.bold,
                color: AppColors.primary,
              ),
            ),
            FilterChip(
              label: const Text('Any Work Model'),
              selected: _anyWorkModel,
              onSelected: (val) {
                setState(() {
                  _anyWorkModel = val;
                });
              },
            ),
          ],
        ),
        if (!_anyWorkModel) ...[
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            children: _availableWorkModels.map((wm) {
              final isSelected = _selectedWorkModels.contains(wm);
              return FilterChip(
                label: Text(wm.toUpperCase()),
                selected: isSelected,
                onSelected: (val) {
                  setState(() {
                    if (val) {
                      _selectedWorkModels.add(wm);
                    } else {
                      _selectedWorkModels.remove(wm);
                    }
                  });
                },
              );
            }).toList(),
          ),
        ],
      ],
    );
  }
}
