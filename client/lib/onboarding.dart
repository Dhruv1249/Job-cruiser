import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:file_picker/file_picker.dart';
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
  late TextEditingController _githubUsernameController;
  late TextEditingController _githubRepoController;
  late TextEditingController _overleafTokenController;

  bool _anyRole = true;
  bool _anyIndustry = true;
  bool _anySalary = true;
  bool _anyWorkModel = true;

  double _minSalary = 120.0;
  final Set<String> _selectedRoles = {'Backend Engineer', 'Fullstack SDE'};
  final Set<String> _selectedIndustries = {'Fintech', 'AI / ML', 'Enterprise SaaS'};
  final Set<String> _selectedWorkModels = {'remote', 'hybrid'};

  @override
  void initState() {
    super.initState();
    _nameController = TextEditingController(text: widget.suggestedName ?? '');
    _bioTextController = TextEditingController();
    _overleafUrlController = TextEditingController();
    _githubUsernameController = TextEditingController();
    _githubRepoController = TextEditingController();
    _overleafTokenController = TextEditingController();
  }

  @override
  void dispose() {
    _nameController.dispose();
    _bioTextController.dispose();
    _overleafUrlController.dispose();
    _githubUsernameController.dispose();
    _githubRepoController.dispose();
    _overleafTokenController.dispose();
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
        final rawStr = latin1.decode(file.bytes!);
        final matches = RegExp(r'[\x20-\x7E\t\r\n]{4,}').allMatches(rawStr);
        final lines = matches
            .map((m) => m.group(0)!.trim())
            .where((s) =>
                !s.startsWith('/') &&
                !s.contains('obj') &&
                !s.contains('endobj') &&
                !s.contains('stream') &&
                s.length > 3)
            .toList();
        extractedText = lines.join('\n');
      }

      if (extractedText.trim().isEmpty) {
        extractedText = "Parsed experience summary from ${file.name}";
      }

      setState(() {
        _bioTextController.text = extractedText.trim();
      });

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('Extracted resume text from ${file.name}!')),
      );
    } catch (e) {
      if (!mounted) return;
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
    final workModels = _anyWorkModel ? ['any'] : _selectedWorkModels.toList();
    final minSalaryVal = _anySalary ? 0 : (_minSalary.toInt() * 1000);

    final prefSuccess = await _apiService.savePreferences({
      'full_name': _nameController.text.trim().isNotEmpty
          ? _nameController.text.trim()
          : 'User',
      'target_roles': targetRoles,
      'target_industries': targetIndustries,
      'work_models': workModels,
      'min_salary': minSalaryVal,
      'currency': 'USD',
      'bio_experience_text': _bioTextController.text.trim(),
    });

    if (_overleafUrlController.text.trim().isNotEmpty) {
      await _apiService.saveOverleafConfig(
        deploymentUrl: _overleafUrlController.text.trim(),
        githubUsername: _githubUsernameController.text.trim(),
        githubRepoName: _githubRepoController.text.trim(),
        accessToken: _overleafTokenController.text.trim(),
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
        const SizedBox(height: 12),
        TextField(
          controller: _nameController,
          decoration: const InputDecoration(
            labelText: 'Full Name',
            hintText: 'Enter your full name',
            border: OutlineInputBorder(),
            contentPadding: EdgeInsets.symmetric(horizontal: 16, vertical: 16),
          ),
        ),
        const SizedBox(height: 24),
        const Text(
          'Self-Hosted open-overleaf Configuration (Optional)',
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
            labelText: 'Deployment Base URL (e.g. https://overleaf.domain.com)',
            border: OutlineInputBorder(),
            isDense: true,
          ),
        ),
        const SizedBox(height: 8),
        Row(
          children: [
            Expanded(
              child: TextField(
                controller: _githubUsernameController,
                decoration: const InputDecoration(
                  labelText: 'GitHub Username',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: TextField(
                controller: _githubRepoController,
                decoration: const InputDecoration(
                  labelText: 'GitHub Repo Name',
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 8),
        TextField(
          controller: _overleafTokenController,
          obscureText: true,
          decoration: const InputDecoration(
            labelText: 'GitHub Personal Access Token / Overleaf API Key',
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
        OutlinedButton.icon(
          onPressed: _extractPdfToBio,
          icon: const Icon(Icons.picture_as_pdf, color: AppColors.primary),
          label: const Text('Import & Parse Text from PDF Resume'),
          style: OutlinedButton.styleFrom(
            foregroundColor: AppColors.primary,
            side: const BorderSide(color: AppColors.outlineVariant),
            padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 16),
          ),
        ),
        const SizedBox(height: 16),
        const Text(
          'Unified Bio & Experience Summary',
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.bold,
            color: AppColors.primary,
          ),
        ),
        const SizedBox(height: 6),
        TextField(
          controller: _bioTextController,
          maxLines: 8,
          decoration: const InputDecoration(
            hintText:
                'Paste or edit your key achievements, skills, and past experience summary here...',
            border: OutlineInputBorder(),
          ),
        ),
      ],
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
        const SizedBox(height: 12),
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
        const SizedBox(height: 12),
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
        const SizedBox(height: 12),
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
      ],
    );
  }
}
