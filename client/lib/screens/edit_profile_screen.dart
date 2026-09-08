import "dart:convert";
import "package:file_picker/file_picker.dart";
import "package:flutter/material.dart";
import "package:syncfusion_flutter_pdf/pdf.dart";
import "../main.dart" show AppColors;
import "../services/api_service.dart";

/// Screen enabling users to view and update their personal profile, contact links, bio, and structured resume records.
class EditProfileScreen extends StatefulWidget {
  const EditProfileScreen({
    super.key,
    this.initialProfileData,
  });

  final Map<String, dynamic>? initialProfileData;

  @override
  State<EditProfileScreen> createState() => _EditProfileScreenState();
}

class _EditProfileScreenState extends State<EditProfileScreen> {
  final ApiService _apiService = ApiService();
  final GlobalKey<FormState> _formKey = GlobalKey<FormState>();

  late final TextEditingController _fullNameController;
  late final TextEditingController _emailController;
  late final TextEditingController _phoneController;
  late final TextEditingController _locationController;
  late final TextEditingController _countryController;
  late final TextEditingController _linkedinController;
  late final TextEditingController _githubController;
  late final TextEditingController _portfolioController;
  late final TextEditingController _bioTextController;
  late final TextEditingController _newSkillController;

  final List<Map<String, TextEditingController>> _customLinkControllers = [];

  List<String> _skills = [];
  List<Map<String, dynamic>> _projects = [];
  List<Map<String, dynamic>> _experiences = [];
  List<Map<String, dynamic>> _education = [];
  List<Map<String, dynamic>> _achievements = [];
  List<Map<String, dynamic>> _certifications = [];

  bool _isLoading = false;
  bool _isSaving = false;
  bool _isParsingCV = false;

  @override
  void initState() {
    super.initState();
    _fullNameController = TextEditingController();
    _emailController = TextEditingController();
    _phoneController = TextEditingController();
    _locationController = TextEditingController();
    _countryController = TextEditingController();
    _linkedinController = TextEditingController();
    _githubController = TextEditingController();
    _portfolioController = TextEditingController();
    _bioTextController = TextEditingController();
    _newSkillController = TextEditingController();

    if (widget.initialProfileData != null) {
      _populateFromData(widget.initialProfileData!);
    }
    if (_bioTextController.text.trim().isEmpty || widget.initialProfileData == null) {
      _loadProfileData();
    }
  }

  @override
  void dispose() {
    _fullNameController.dispose();
    _emailController.dispose();
    _phoneController.dispose();
    _locationController.dispose();
    _countryController.dispose();
    _linkedinController.dispose();
    _githubController.dispose();
    _portfolioController.dispose();
    _bioTextController.dispose();
    _newSkillController.dispose();
    for (final linkEntry in _customLinkControllers) {
      linkEntry["label"]?.dispose();
      linkEntry["url"]?.dispose();
    }
    super.dispose();
  }

  void _populateFromData(Map<String, dynamic> data) {
    _fullNameController.text = data["full_name"] as String? ?? "";
    _emailController.text = data["email"] as String? ?? "";
    _phoneController.text = data["phone"] as String? ?? "";
    _locationController.text = data["location"] as String? ?? "";
    _countryController.text = data["country"] as String? ?? "";
    _linkedinController.text = data["linkedin_url"] as String? ?? "";
    _githubController.text = data["github_url"] as String? ?? "";
    _portfolioController.text = data["portfolio_url"] as String? ?? "";
    String bioText = data["bio_experience_text"] as String? ?? data["bio_summary"] as String? ?? "";
    if (bioText.trim().isEmpty && data["master_cv_text"] != null) {
      final masterCv = data["master_cv_text"].toString();
      final delimiterIndex = masterCv.indexOf("--- STRUCTURED RESUME DETAILS ---");
      if (delimiterIndex != -1) {
        bioText = masterCv.substring(0, delimiterIndex).trim();
      } else {
        bioText = masterCv.trim();
      }
    }
    _bioTextController.text = bioText;

    final customLinksList = data["custom_links"] as List<dynamic>? ?? [];
    _customLinkControllers.clear();
    for (final rawItem in customLinksList) {
      if (rawItem is Map) {
        final label = rawItem["label"]?.toString() ?? "";
        final url = rawItem["url"]?.toString() ?? "";
        _customLinkControllers.add({
          "label": TextEditingController(text: label),
          "url": TextEditingController(text: url),
        });
      }
    }

    final rawSkills = data["skills"] as List<dynamic>? ?? [];
    _skills = rawSkills.map((item) => item.toString()).where((s) => s.isNotEmpty).toList();

    final rawProjects = data["projects"] as List<dynamic>? ?? [];
    _projects = rawProjects.whereType<Map>().map((item) {
      final itemMap = Map<String, dynamic>.from(item);
      final rawTechStack = itemMap["tech_stack"];
      List<String> parsedTechStack = [];
      if (rawTechStack is List) {
        parsedTechStack = rawTechStack.map((techItem) => techItem.toString()).toList();
      } else if (rawTechStack is String && rawTechStack.isNotEmpty) {
        parsedTechStack = rawTechStack.split(",").map((techItem) => techItem.trim()).toList();
      }
      itemMap["tech_stack"] = parsedTechStack;
      return itemMap;
    }).toList();

    final rawExperiences = data["experiences"] as List<dynamic>? ?? [];
    _experiences = rawExperiences.whereType<Map>().map((item) => Map<String, dynamic>.from(item)).toList();

    final rawEducation = data["education"] as List<dynamic>? ?? [];
    _education = rawEducation.whereType<Map>().map((item) => Map<String, dynamic>.from(item)).toList();

    final rawAchievements = data["achievements"] as List<dynamic>? ?? [];
    _achievements = rawAchievements.whereType<Map>().map((item) => Map<String, dynamic>.from(item)).toList();

    final rawCertifications = data["certifications"] as List<dynamic>? ?? [];
    _certifications = rawCertifications.whereType<Map>().map((item) => Map<String, dynamic>.from(item)).toList();
  }

  Future<void> _loadProfileData() async {
    setState(() => _isLoading = true);
    final preferencesData = await _apiService.fetchPreferences();
    if (!mounted) return;
    if (preferencesData != null) {
      setState(() {
        _populateFromData(preferencesData);
        _isLoading = false;
      });
    } else {
      setState(() => _isLoading = false);
    }
  }

  void _addCustomLink({String label = "", String url = ""}) {
    setState(() {
      _customLinkControllers.add({
        "label": TextEditingController(text: label),
        "url": TextEditingController(text: url),
      });
    });
  }

  void _removeCustomLink(int index) {
    setState(() {
      _customLinkControllers[index]["label"]?.dispose();
      _customLinkControllers[index]["url"]?.dispose();
      _customLinkControllers.removeAt(index);
    });
  }

  void _addSkill() {
    final text = _newSkillController.text.trim();
    if (text.isNotEmpty && !_skills.contains(text)) {
      setState(() {
        _skills.add(text);
        _newSkillController.clear();
      });
    }
  }

  void _removeSkill(String skill) {
    setState(() {
      _skills.remove(skill);
    });
  }

  Future<void> _showProjectDialog({Map<String, dynamic>? existingProject, int? editIndex}) async {
    final titleController = TextEditingController(text: existingProject?["title"]?.toString() ?? "");
    final techStackController = TextEditingController(
      text: existingProject?["tech_stack"] is List
          ? (existingProject!["tech_stack"] as List).join(", ")
          : existingProject?["tech_stack"]?.toString() ?? "",
    );
    final descriptionController = TextEditingController(text: existingProject?["description"]?.toString() ?? "");
    final linkController = TextEditingController(text: existingProject?["link"]?.toString() ?? "");

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(existingProject != null ? "Edit Project" : "Add Project"),
        content: SingleChildScrollView(
          child: SizedBox(
            width: 480,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: titleController,
                  decoration: const InputDecoration(
                    labelText: "Project Title *",
                    hintText: "e.g. Distributed Task Queue",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: techStackController,
                  decoration: const InputDecoration(
                    labelText: "Tech Stack (comma separated)",
                    hintText: "Go, Docker, Redis, gRPC",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: descriptionController,
                  maxLines: 3,
                  decoration: const InputDecoration(
                    labelText: "Description",
                    hintText: "Engineered a scalable queue handling 10k tasks/sec...",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: linkController,
                  decoration: const InputDecoration(
                    labelText: "Project / GitHub Link",
                    hintText: "https://github.com/...",
                  ),
                ),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            onPressed: () {
              if (titleController.text.trim().isEmpty) return;
              FocusManager.instance.primaryFocus?.unfocus();
              Navigator.of(dialogContext).pop(true);
            },
            child: const Text("Save"),
          ),
        ],
      ),
    );

    if (result == true && mounted) {
      final techStack = techStackController.text
          .split(",")
          .map((tech) => tech.trim())
          .where((tech) => tech.isNotEmpty)
          .toList();

      final updatedItem = <String, dynamic>{
        "title": titleController.text.trim(),
        "tech_stack": techStack,
        "description": descriptionController.text.trim(),
        "link": linkController.text.trim(),
      };

      setState(() {
        if (editIndex != null && editIndex >= 0 && editIndex < _projects.length) {
          _projects[editIndex] = updatedItem;
        } else {
          _projects.add(updatedItem);
        }
      });
    }

    titleController.dispose();
    techStackController.dispose();
    descriptionController.dispose();
    linkController.dispose();
  }

  Future<void> _showExperienceDialog({Map<String, dynamic>? existingExperience, int? editIndex}) async {
    final companyController = TextEditingController(text: existingExperience?["company"]?.toString() ?? "");
    final roleController = TextEditingController(text: existingExperience?["role"]?.toString() ?? "");
    final durationController = TextEditingController(text: existingExperience?["duration"]?.toString() ?? "");
    final highlightsController = TextEditingController(
      text: existingExperience?["highlights"] is List
          ? (existingExperience!["highlights"] as List).join("\n")
          : existingExperience?["highlights"]?.toString() ?? "",
    );

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(existingExperience != null ? "Edit Experience" : "Add Experience"),
        content: SingleChildScrollView(
          child: SizedBox(
            width: 480,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: companyController,
                  decoration: const InputDecoration(
                    labelText: "Company *",
                    hintText: "e.g. Google or Startup Inc.",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: roleController,
                  decoration: const InputDecoration(
                    labelText: "Role Title *",
                    hintText: "e.g. Senior Software Engineer",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: durationController,
                  decoration: const InputDecoration(
                    labelText: "Duration",
                    hintText: "e.g. Jan 2023 - Present",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: highlightsController,
                  maxLines: 4,
                  decoration: const InputDecoration(
                    labelText: "Key Highlights (one per line)",
                    hintText: "Architected microservices...\nImproved query latency by 45%...",
                  ),
                ),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            onPressed: () {
              if (companyController.text.trim().isEmpty || roleController.text.trim().isEmpty) return;
              FocusManager.instance.primaryFocus?.unfocus();
              Navigator.of(dialogContext).pop(true);
            },
            child: const Text("Save"),
          ),
        ],
      ),
    );

    if (result == true && mounted) {
      final updatedItem = <String, dynamic>{
        "company": companyController.text.trim(),
        "role": roleController.text.trim(),
        "duration": durationController.text.trim(),
        "highlights": highlightsController.text.trim(),
      };

      setState(() {
        if (editIndex != null && editIndex >= 0 && editIndex < _experiences.length) {
          _experiences[editIndex] = updatedItem;
        } else {
          _experiences.add(updatedItem);
        }
      });
    }

    companyController.dispose();
    roleController.dispose();
    durationController.dispose();
    highlightsController.dispose();
  }

  Future<void> _showEducationDialog({Map<String, dynamic>? existingEducation, int? editIndex}) async {
    final institutionController = TextEditingController(text: existingEducation?["institution"]?.toString() ?? "");
    final degreeController = TextEditingController(text: existingEducation?["degree"]?.toString() ?? "");
    final yearController = TextEditingController(text: existingEducation?["year"]?.toString() ?? "");
    final gradeController = TextEditingController(text: existingEducation?["grade"]?.toString() ?? "");

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text(existingEducation != null ? "Edit Education" : "Add Education"),
        content: SingleChildScrollView(
          child: SizedBox(
            width: 480,
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: institutionController,
                  decoration: const InputDecoration(
                    labelText: "Institution / University *",
                    hintText: "e.g. Stanford University",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: degreeController,
                  decoration: const InputDecoration(
                    labelText: "Degree / Major *",
                    hintText: "e.g. B.S. in Computer Science",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: yearController,
                  decoration: const InputDecoration(
                    labelText: "Graduation Year",
                    hintText: "e.g. 2020 - 2024",
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: gradeController,
                  decoration: const InputDecoration(
                    labelText: "GPA / Grade",
                    hintText: "e.g. 3.9 / 4.0 or 8.8 CGPA",
                  ),
                ),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            onPressed: () {
              if (institutionController.text.trim().isEmpty || degreeController.text.trim().isEmpty) return;
              FocusManager.instance.primaryFocus?.unfocus();
              Navigator.of(dialogContext).pop(true);
            },
            child: const Text("Save"),
          ),
        ],
      ),
    );

    if (result == true && mounted) {
      final updatedItem = <String, dynamic>{
        "institution": institutionController.text.trim(),
        "degree": degreeController.text.trim(),
        "year": yearController.text.trim(),
        "grade": gradeController.text.trim(),
      };

      setState(() {
        if (editIndex != null && editIndex >= 0 && editIndex < _education.length) {
          _education[editIndex] = updatedItem;
        } else {
          _education.add(updatedItem);
        }
      });
    }

    institutionController.dispose();
    degreeController.dispose();
    yearController.dispose();
    gradeController.dispose();
  }

  Future<void> _extractPdfToBio() async {
    try {
      final result = await FilePicker.platform.pickFiles(
        type: FileType.custom,
        allowedExtensions: ["pdf", "txt"],
        withData: true,
      );

      if (result == null || result.files.isEmpty) {
        return;
      }

      final file = result.files.first;
      if (!mounted) return;

      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("Reading ${file.name}...")),
      );

      String extractedText = "";
      if (file.bytes != null) {
        if (file.name.toLowerCase().endsWith(".pdf")) {
          final PdfDocument document = PdfDocument(inputBytes: file.bytes!);
          extractedText = PdfTextExtractor(document).extractText();
          document.dispose();
        } else {
          extractedText = utf8.decode(file.bytes!, allowMalformed: true);
        }

        final cleanLines = extractedText
            .split("\n")
            .map((line) => line.trim())
            .where((line) => line.isNotEmpty && !line.startsWith("%PDF-"))
            .toList();
        extractedText = cleanLines.join("\n");
      }

      if (extractedText.trim().isEmpty) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text("No readable text found in selected file.")),
        );
        return;
      }

      setState(() {
        _isParsingCV = true;
      });

      final parsed = await _apiService.parseCVWithGemini(extractedText);
      if (!mounted) return;

      setState(() {
        _isParsingCV = false;
        if (parsed != null) {
          if (parsed["bio_summary"] != null && parsed["bio_summary"].toString().isNotEmpty) {
            _bioTextController.text = parsed["bio_summary"].toString();
          }
          if (parsed["location"] != null && _locationController.text.trim().isEmpty) {
            _locationController.text = parsed["location"].toString();
          }
          if (parsed["skills"] is List) {
            final parsedSkillsList = List<String>.from(parsed["skills"]);
            for (final skillItem in parsedSkillsList) {
              if (!_skills.contains(skillItem)) {
                _skills.add(skillItem);
              }
            }
          }
          if (parsed["projects"] is List) {
            final parsedProjectsList = (parsed["projects"] as List)
                .whereType<Map>()
                .map((m) => Map<String, dynamic>.from(m))
                .toList();
            _projects.addAll(parsedProjectsList);
          }
          if (parsed["experience"] is List) {
            final parsedExpList = (parsed["experience"] as List)
                .whereType<Map>()
                .map((m) => Map<String, dynamic>.from(m))
                .toList();
            _experiences.addAll(parsedExpList);
          }
          if (parsed["education"] is List) {
            final parsedEduList = (parsed["education"] as List)
                .whereType<Map>()
                .map((m) => Map<String, dynamic>.from(m))
                .toList();
            _education.addAll(parsedEduList);
          }
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text("CV parsed and structured profile details updated!"),
              backgroundColor: AppColors.successGreen,
            ),
          );
        } else {
          _bioTextController.text = extractedText;
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(content: Text("Extracted raw text to summary.")),
          );
        }
      });
    } catch (error) {
      if (!mounted) return;
      setState(() => _isParsingCV = false);
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text("Error reading file: $error")),
      );
    }
  }

  Future<void> _handleSaveProfile() async {
    if (!_formKey.currentState!.validate()) {
      return;
    }

    setState(() => _isSaving = true);

    final customLinksPayload = _customLinkControllers
        .map((entry) => {
              "label": entry["label"]?.text.trim() ?? "",
              "url": entry["url"]?.text.trim() ?? "",
            })
        .where((entry) => entry["url"]!.isNotEmpty)
        .toList();

    final profilePayload = <String, dynamic>{
      "full_name": _fullNameController.text.trim().isNotEmpty
          ? _fullNameController.text.trim()
          : "User",
      "email": _emailController.text.trim(),
      "phone": _phoneController.text.trim(),
      "location": _locationController.text.trim(),
      "country": _countryController.text.trim(),
      "linkedin_url": _linkedinController.text.trim(),
      "github_url": _githubController.text.trim(),
      "portfolio_url": _portfolioController.text.trim(),
      "custom_links": customLinksPayload,
      "bio_summary": _bioTextController.text.trim(),
      "bio_experience_text": _bioTextController.text.trim(),
      "skills": _skills,
      "projects": _projects,
      "experiences": _experiences,
      "education": _education,
      "achievements": _achievements,
      "certifications": _certifications,
    };

    final success = await _apiService.saveProfile(profilePayload);

    if (!mounted) return;
    setState(() => _isSaving = false);

    if (success) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Profile details saved successfully!"),
          backgroundColor: AppColors.successGreen,
        ),
      );
      Navigator.of(context).pop(true);
    } else {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Failed to save profile. Please try again."),
          backgroundColor: AppColors.error,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        backgroundColor: AppColors.surface,
        elevation: 0,
        scrolledUnderElevation: 0,
        title: const Text(
          "Edit Profile & Experience",
          style: TextStyle(
            color: AppColors.primary,
            fontWeight: FontWeight.bold,
            fontSize: 20,
          ),
        ),
        actions: [
          Padding(
            padding: const EdgeInsets.only(right: 16),
            child: ElevatedButton.icon(
              onPressed: _isSaving ? null : _handleSaveProfile,
              icon: _isSaving
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                    )
                  : const Icon(Icons.check, size: 18),
              label: const Text("Save Profile"),
              style: ElevatedButton.styleFrom(
                backgroundColor: AppColors.primary,
                foregroundColor: Colors.white,
              ),
            ),
          ),
        ],
      ),
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : Form(
              key: _formKey,
              child: SingleChildScrollView(
                padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
                child: Center(
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 768),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        _buildSectionHeader(
                          title: "Personal Information",
                          subtitle: "Your primary contact details and identity.",
                          icon: Icons.person_outline,
                        ),
                        const SizedBox(height: 12),
                        _buildPersonalInfoCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Professional Links",
                          subtitle: "Public profiles and project repositories.",
                          icon: Icons.link,
                        ),
                        const SizedBox(height: 12),
                        _buildLinksCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Professional Summary",
                          subtitle: "First-person bio summary used by AI to tailor resumes and cover letters.",
                          icon: Icons.article_outlined,
                          trailing: OutlinedButton.icon(
                            onPressed: _isParsingCV ? null : _extractPdfToBio,
                            icon: _isParsingCV
                                ? const SizedBox(
                                    width: 14,
                                    height: 14,
                                    child: CircularProgressIndicator(strokeWidth: 2),
                                  )
                                : const Icon(Icons.upload_file, size: 16),
                            label: const Text("Parse from CV"),
                          ),
                        ),
                        const SizedBox(height: 12),
                        _buildBioCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Key Technical Skills",
                          subtitle: "Core languages, frameworks, databases, and tooling.",
                          icon: Icons.code,
                        ),
                        const SizedBox(height: 12),
                        _buildSkillsCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Featured Projects",
                          subtitle: "Key technical projects, architecture highlights, and repositories.",
                          icon: Icons.rocket_launch_outlined,
                          trailing: OutlinedButton.icon(
                            onPressed: () => _showProjectDialog(),
                            icon: const Icon(Icons.add, size: 16),
                            label: const Text("Add Project"),
                          ),
                        ),
                        const SizedBox(height: 12),
                        _buildProjectsCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Work Experience",
                          subtitle: "Professional career history, leadership, and accomplishments.",
                          icon: Icons.business_center_outlined,
                          trailing: OutlinedButton.icon(
                            onPressed: () => _showExperienceDialog(),
                            icon: const Icon(Icons.add, size: 16),
                            label: const Text("Add Experience"),
                          ),
                        ),
                        const SizedBox(height: 12),
                        _buildExperienceCard(),
                        const SizedBox(height: 24),
                        _buildSectionHeader(
                          title: "Education & Credentials",
                          subtitle: "Degrees, institutions, honors, and certifications.",
                          icon: Icons.school_outlined,
                          trailing: OutlinedButton.icon(
                            onPressed: () => _showEducationDialog(),
                            icon: const Icon(Icons.add, size: 16),
                            label: const Text("Add Education"),
                          ),
                        ),
                        const SizedBox(height: 12),
                        _buildEducationCard(),
                        const SizedBox(height: 32),
                        SizedBox(
                          width: double.infinity,
                          height: 50,
                          child: ElevatedButton.icon(
                            onPressed: _isSaving ? null : _handleSaveProfile,
                            icon: _isSaving
                                ? const SizedBox(
                                    width: 20,
                                    height: 20,
                                    child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                  )
                                : const Icon(Icons.save_outlined),
                            label: const Text(
                              "Save All Profile Changes",
                              style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
                            ),
                            style: ElevatedButton.styleFrom(
                              backgroundColor: AppColors.primary,
                              foregroundColor: Colors.white,
                              shape: RoundedRectangleBorder(
                                borderRadius: BorderRadius.circular(10),
                              ),
                            ),
                          ),
                        ),
                        const SizedBox(height: 40),
                      ],
                    ),
                  ),
                ),
              ),
            ),
    );
  }

  Widget _buildSectionHeader({
    required String title,
    required String subtitle,
    required IconData icon,
    Widget? trailing,
  }) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
        Expanded(
          child: Row(
            children: [
              Icon(icon, size: 20, color: AppColors.primary),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.bold,
                        color: AppColors.primary,
                      ),
                    ),
                    Text(
                      subtitle,
                      style: const TextStyle(
                        fontSize: 12,
                        color: AppColors.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
        if (trailing != null) ...[
          const SizedBox(width: 8),
          trailing,
        ],
      ],
    );
  }

  Widget _buildPersonalInfoCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        children: [
          TextFormField(
            controller: _fullNameController,
            decoration: const InputDecoration(
              labelText: "Full Name *",
              prefixIcon: Icon(Icons.person_outline, size: 20),
            ),
            validator: (value) => (value == null || value.trim().isEmpty) ? "Please enter your name" : null,
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _emailController,
            decoration: const InputDecoration(
              labelText: "Email Address",
              prefixIcon: Icon(Icons.email_outlined, size: 20),
            ),
          ),
          const SizedBox(height: 12),
          Row(
            children: [
              Expanded(
                child: TextFormField(
                  controller: _phoneController,
                  decoration: const InputDecoration(
                    labelText: "Phone Number",
                    prefixIcon: Icon(Icons.phone_outlined, size: 20),
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: TextFormField(
                  controller: _locationController,
                  decoration: const InputDecoration(
                    labelText: "Location / City",
                    hintText: "e.g. San Francisco, CA",
                    prefixIcon: Icon(Icons.location_city_outlined, size: 20),
                  ),
                ),
              ),
            ],
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _countryController,
            decoration: const InputDecoration(
              labelText: "Country",
              hintText: "e.g. India, United States, United Kingdom",
              prefixIcon: Icon(Icons.public_outlined, size: 20),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildLinksCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextFormField(
            controller: _linkedinController,
            decoration: const InputDecoration(
              labelText: "LinkedIn URL",
              hintText: "https://linkedin.com/in/...",
              prefixIcon: Icon(Icons.business, size: 20),
            ),
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _githubController,
            decoration: const InputDecoration(
              labelText: "GitHub URL",
              hintText: "https://github.com/...",
              prefixIcon: Icon(Icons.code, size: 20),
            ),
          ),
          const SizedBox(height: 12),
          TextFormField(
            controller: _portfolioController,
            decoration: const InputDecoration(
              labelText: "Portfolio Website",
              hintText: "https://...",
              prefixIcon: Icon(Icons.public, size: 20),
            ),
          ),
          if (_customLinkControllers.isNotEmpty) ...[
            const SizedBox(height: 16),
            const Text(
              "Custom Links",
              style: TextStyle(fontWeight: FontWeight.w600, fontSize: 13),
            ),
            const SizedBox(height: 8),
            for (int i = 0; i < _customLinkControllers.length; i++)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Row(
                  children: [
                    SizedBox(
                      width: 120,
                      child: TextFormField(
                        controller: _customLinkControllers[i]["label"],
                        decoration: const InputDecoration(
                          hintText: "Label (e.g. Blog)",
                          isDense: true,
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: TextFormField(
                        controller: _customLinkControllers[i]["url"],
                        decoration: const InputDecoration(
                          hintText: "https://...",
                          isDense: true,
                        ),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close, size: 18, color: AppColors.error),
                      onPressed: () => _removeCustomLink(i),
                    ),
                  ],
                ),
              ),
          ],
          const SizedBox(height: 8),
          TextButton.icon(
            onPressed: () => _addCustomLink(),
            icon: const Icon(Icons.add, size: 16),
            label: const Text("Add Custom Link"),
          ),
        ],
      ),
    );
  }

  Widget _buildBioCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextFormField(
            controller: _bioTextController,
            maxLines: 5,
            decoration: const InputDecoration(
              hintText: "I am a backend engineer specializing in distributed systems, Go, and cloud architectures...",
              border: OutlineInputBorder(),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildSkillsCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _newSkillController,
                  decoration: const InputDecoration(
                    hintText: "Enter skill (e.g. Go, PostgreSQL, Kubernetes)",
                    isDense: true,
                  ),
                  onSubmitted: (_) => _addSkill(),
                ),
              ),
              const SizedBox(width: 8),
              ElevatedButton.icon(
                onPressed: _addSkill,
                icon: const Icon(Icons.add, size: 16),
                label: const Text("Add"),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (_skills.isEmpty)
            const Text(
              "No skills added yet. Add skills or parse from CV.",
              style: TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
            )
          else
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: _skills
                  .map(
                    (skill) => Chip(
                      label: Text(skill),
                      onDeleted: () => _removeSkill(skill),
                      backgroundColor: AppColors.surfaceContainer,
                      side: const BorderSide(color: AppColors.outlineVariant),
                    ),
                  )
                  .toList(),
            ),
        ],
      ),
    );
  }

  Widget _buildProjectsCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: _projects.isEmpty
          ? const Center(
              child: Padding(
                padding: EdgeInsets.symmetric(vertical: 16),
                child: Text(
                  "No projects added yet. Click \"Add Project\" to highlight your work.",
                  style: TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
                ),
              ),
            )
          : ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: _projects.length,
              separatorBuilder: (separatorContext, separatorIndex) => const Divider(height: 20, color: AppColors.outlineVariant),
              itemBuilder: (context, index) {
                final project = _projects[index];
                final title = project["title"]?.toString() ?? "Untitled";
                final description = project["description"]?.toString() ?? "";
                final link = project["link"]?.toString() ?? "";
                final techStack = project["tech_stack"] is List
                    ? List<String>.from(project["tech_stack"])
                    : <String>[];

                return Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            title,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
                          ),
                          if (description.isNotEmpty) ...[
                            const SizedBox(height: 4),
                            Text(
                              description,
                              style: const TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
                            ),
                          ],
                          if (techStack.isNotEmpty) ...[
                            const SizedBox(height: 6),
                            Wrap(
                              spacing: 6,
                              runSpacing: 4,
                              children: techStack
                                  .map(
                                    (tech) => Container(
                                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                                      decoration: BoxDecoration(
                                        color: AppColors.surfaceContainerHigh,
                                        borderRadius: BorderRadius.circular(4),
                                      ),
                                      child: Text(
                                        tech,
                                        style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w500),
                                      ),
                                    ),
                                  )
                                  .toList(),
                            ),
                          ],
                          if (link.isNotEmpty) ...[
                            const SizedBox(height: 4),
                            Text(
                              link,
                              style: const TextStyle(color: AppColors.primary, fontSize: 12),
                            ),
                          ],
                        ],
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.edit_outlined, size: 18),
                      onPressed: () => _showProjectDialog(existingProject: project, editIndex: index),
                    ),
                    IconButton(
                      icon: const Icon(Icons.delete_outline, size: 18, color: AppColors.error),
                      onPressed: () => setState(() => _projects.removeAt(index)),
                    ),
                  ],
                );
              },
            ),
    );
  }

  Widget _buildExperienceCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: _experiences.isEmpty
          ? const Center(
              child: Padding(
                padding: EdgeInsets.symmetric(vertical: 16),
                child: Text(
                  "No experience records added yet. Click \"Add Experience\" to record your history.",
                  style: TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
                ),
              ),
            )
          : ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: _experiences.length,
              separatorBuilder: (separatorContext, separatorIndex) => const Divider(height: 20, color: AppColors.outlineVariant),
              itemBuilder: (context, index) {
                final exp = _experiences[index];
                final company = exp["company"]?.toString() ?? "Company";
                final role = exp["role"]?.toString() ?? "Role";
                final duration = exp["duration"]?.toString() ?? "";
                final highlights = exp["highlights"]?.toString() ?? "";

                return Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            role,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 15),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            duration.isNotEmpty ? "$company • $duration" : company,
                            style: const TextStyle(
                              color: AppColors.onSurfaceVariant,
                              fontSize: 13,
                              fontWeight: FontWeight.w500,
                            ),
                          ),
                          if (highlights.isNotEmpty) ...[
                            const SizedBox(height: 6),
                            Text(
                              highlights,
                              style: const TextStyle(color: AppColors.onSurfaceVariant, fontSize: 12),
                            ),
                          ],
                        ],
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.edit_outlined, size: 18),
                      onPressed: () => _showExperienceDialog(existingExperience: exp, editIndex: index),
                    ),
                    IconButton(
                      icon: const Icon(Icons.delete_outline, size: 18, color: AppColors.error),
                      onPressed: () => setState(() => _experiences.removeAt(index)),
                    ),
                  ],
                );
              },
            ),
    );
  }

  Widget _buildEducationCard() {
    return Container(
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: AppColors.outlineVariant),
      ),
      child: _education.isEmpty
          ? const Center(
              child: Padding(
                padding: EdgeInsets.symmetric(vertical: 16),
                child: Text(
                  "No education records added yet.",
                  style: TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
                ),
              ),
            )
          : ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: _education.length,
              separatorBuilder: (separatorContext, separatorIndex) => const Divider(height: 20, color: AppColors.outlineVariant),
              itemBuilder: (context, index) {
                final edu = _education[index];
                final institution = edu["institution"]?.toString() ?? "";
                final degree = edu["degree"]?.toString() ?? "";
                final year = edu["year"]?.toString() ?? "";
                final grade = edu["grade"]?.toString() ?? "";

                return Row(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            degree,
                            style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 14),
                          ),
                          const SizedBox(height: 2),
                          Text(
                            institution,
                            style: const TextStyle(color: AppColors.onSurfaceVariant, fontSize: 13),
                          ),
                          if (year.isNotEmpty || grade.isNotEmpty) ...[
                            const SizedBox(height: 2),
                            Text(
                              [if (year.isNotEmpty) year, if (grade.isNotEmpty) grade].join(" • "),
                              style: const TextStyle(color: AppColors.outline, fontSize: 12),
                            ),
                          ],
                        ],
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.edit_outlined, size: 18),
                      onPressed: () => _showEducationDialog(existingEducation: edu, editIndex: index),
                    ),
                    IconButton(
                      icon: const Icon(Icons.delete_outline, size: 18, color: AppColors.error),
                      onPressed: () => setState(() => _education.removeAt(index)),
                    ),
                  ],
                );
              },
            ),
    );
  }
}
