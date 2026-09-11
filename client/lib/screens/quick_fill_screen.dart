import "dart:async";
import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "../main.dart" show AppColors;
import "../models/quick_fill_item.dart";
import "../services/api_service.dart";

/// Screen providing single-tap clipboard copy targets for job application forms and custom user answers.
class QuickFillScreen extends StatefulWidget {
  final Map<String, dynamic>? initialProfileData;
  final Future<bool> Function(Map<String, dynamic> payload)? onSavePreferences;

  const QuickFillScreen({
    super.key,
    this.initialProfileData,
    this.onSavePreferences,
  });

  @override
  State<QuickFillScreen> createState() => _QuickFillScreenState();
}

class _QuickFillScreenState extends State<QuickFillScreen> {
  final ApiService _apiService = ApiService();
  final TextEditingController _searchController = TextEditingController();

  bool _isLoading = true;
  String _selectedCategory = "All";
  String _copiedItemId = "";
  Timer? _copiedResetTimer;

  Map<String, dynamic> _profileData = {};
  Map<String, dynamic> _customFormAnswers = {};
  List<Map<String, dynamic>> _customFields = [];
  Map<String, dynamic> _fieldOverrides = {};
  Set<String> _deletedFieldIds = {};

  final List<String> _categoryFilterOptions = const [
    "All",
    "Personal & Contact",
    "Work Authorization",
    "Socials & Links",
    "Experience & Education",
    "Custom Answers",
  ];

  @override
  void initState() {
    super.initState();
    if (widget.initialProfileData != null) {
      _profileData = Map<String, dynamic>.from(widget.initialProfileData!);
      final rawCustomAnswers = _profileData["custom_form_answers"] is Map
          ? Map<String, dynamic>.from(_profileData["custom_form_answers"] as Map)
          : <String, dynamic>{};
      final rawCustomFields = rawCustomAnswers["custom_fields"] is List
          ? (rawCustomAnswers["custom_fields"] as List)
              .whereType<Map>()
              .map((entry) => Map<String, dynamic>.from(entry))
              .toList()
          : <Map<String, dynamic>>[];
      final rawOverrides = rawCustomAnswers["field_overrides"] is Map
          ? Map<String, dynamic>.from(rawCustomAnswers["field_overrides"] as Map)
          : <String, dynamic>{};
      final rawDeletedIds = rawCustomAnswers["deleted_field_ids"] is List
          ? Set<String>.from((rawCustomAnswers["deleted_field_ids"] as List).map((entry) => entry.toString()))
          : <String>{};

      _customFormAnswers = rawCustomAnswers;
      _customFields = rawCustomFields;
      _fieldOverrides = rawOverrides;
      _deletedFieldIds = rawDeletedIds;
      _isLoading = false;
    } else {
      _loadPreferences();
    }
    _searchController.addListener(_onSearchChanged);
  }

  @override
  void dispose() {
    _searchController.removeListener(_onSearchChanged);
    _searchController.dispose();
    _copiedResetTimer?.cancel();
    super.dispose();
  }

  void _onSearchChanged() {
    setState(() {});
  }

  Future<void> _loadPreferences() async {
    setState(() {
      _isLoading = true;
    });

    final data = await _apiService.fetchPreferences();
    if (!mounted) return;

    if (data != null) {
      final rawCustomAnswers = data["custom_form_answers"] is Map
          ? Map<String, dynamic>.from(data["custom_form_answers"] as Map)
          : <String, dynamic>{};

      final rawCustomFields = rawCustomAnswers["custom_fields"] is List
          ? (rawCustomAnswers["custom_fields"] as List)
              .whereType<Map>()
              .map((entry) => Map<String, dynamic>.from(entry))
              .toList()
          : <Map<String, dynamic>>[];

      final rawOverrides = rawCustomAnswers["field_overrides"] is Map
          ? Map<String, dynamic>.from(rawCustomAnswers["field_overrides"] as Map)
          : <String, dynamic>{};

      final rawDeletedIds = rawCustomAnswers["deleted_field_ids"] is List
          ? Set<String>.from((rawCustomAnswers["deleted_field_ids"] as List).map((entry) => entry.toString()))
          : <String>{};

      setState(() {
        _profileData = data;
        _customFormAnswers = rawCustomAnswers;
        _customFields = rawCustomFields;
        _fieldOverrides = rawOverrides;
        _deletedFieldIds = rawDeletedIds;
        _isLoading = false;
      });
    } else {
      setState(() {
        _isLoading = false;
      });
    }
  }

  Future<void> _persistCustomFormAnswers() async {
    final updatedPayload = Map<String, dynamic>.from(_customFormAnswers);
    updatedPayload["custom_fields"] = _customFields;
    updatedPayload["field_overrides"] = _fieldOverrides;
    updatedPayload["deleted_field_ids"] = _deletedFieldIds.toList();

    final Map<String, dynamic> payload = Map<String, dynamic>.from(_profileData);
    payload["custom_form_answers"] = updatedPayload;

    if (widget.onSavePreferences != null) {
      await widget.onSavePreferences!(payload);
    } else {
      await _apiService.savePreferences(payload);
    }
  }

  List<QuickFillItem> _buildAllItems() {
    final List<QuickFillItem> baseItems = [];

    final fullName = (_profileData["full_name"] ?? "").toString().trim();
    if (fullName.isNotEmpty && fullName != "User") {
      baseItems.add(QuickFillItem(
        id: "profile_full_name",
        label: "Full Name",
        value: fullName,
        category: "Personal & Contact",
      ));

      final nameTokens = fullName.split(" ").where((token) => token.trim().isNotEmpty).toList();
      if (nameTokens.isNotEmpty) {
        baseItems.add(QuickFillItem(
          id: "profile_first_name",
          label: "First Name",
          value: nameTokens.first,
          category: "Personal & Contact",
        ));
      }
      if (nameTokens.length > 1) {
        baseItems.add(QuickFillItem(
          id: "profile_last_name",
          label: "Last Name",
          value: nameTokens.sublist(1).join(" "),
          category: "Personal & Contact",
        ));
      }
    }

    final email = (_profileData["email"] ?? "").toString().trim();
    if (email.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "profile_email",
        label: "Email Address",
        value: email,
        category: "Personal & Contact",
      ));
    }

    final phone = (_profileData["phone"] ?? "").toString().trim();
    if (phone.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "profile_phone",
        label: "Phone Number",
        value: phone,
        category: "Personal & Contact",
      ));
    }

    final location = (_profileData["location"] ?? "").toString().trim();
    if (location.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "profile_location",
        label: "Current Location",
        value: location,
        category: "Personal & Contact",
      ));
    }

    final country = (_profileData["country"] ?? "").toString().trim();
    if (country.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "profile_country",
        label: "Country",
        value: country,
        category: "Personal & Contact",
      ));
    }

    final linkedIn = (_profileData["linkedin_url"] ?? "").toString().trim();
    if (linkedIn.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "link_linkedin",
        label: "LinkedIn Profile URL",
        value: linkedIn,
        category: "Socials & Links",
      ));
    }

    final github = (_profileData["github_url"] ?? "").toString().trim();
    if (github.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "link_github",
        label: "GitHub Profile URL",
        value: github,
        category: "Socials & Links",
      ));
    }

    final portfolio = (_profileData["portfolio_url"] ?? "").toString().trim();
    if (portfolio.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "link_portfolio",
        label: "Portfolio / Website URL",
        value: portfolio,
        category: "Socials & Links",
      ));
    }

    if (_profileData["custom_links"] is List) {
      final customLinks = _profileData["custom_links"] as List;
      for (int index = 0; index < customLinks.length; index++) {
        final linkMap = customLinks[index];
        if (linkMap is Map) {
          final label = linkMap["label"]?.toString().trim() ?? "Custom Link";
          final url = linkMap["url"]?.toString().trim() ?? "";
          if (url.isNotEmpty) {
            baseItems.add(QuickFillItem(
              id: "link_custom_$index",
              label: label,
              value: url,
              category: "Socials & Links",
            ));
          }
        }
      }
    }

    if (_profileData["experiences"] is List && (_profileData["experiences"] as List).isNotEmpty) {
      final firstExperience = (_profileData["experiences"] as List).first;
      if (firstExperience is Map) {
        final role = firstExperience["role"]?.toString().trim() ?? "";
        final company = firstExperience["company"]?.toString().trim() ?? "";
        final duration = firstExperience["duration"]?.toString().trim() ?? "";
        if (role.isNotEmpty && company.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "exp_current_role",
            label: "Current / Most Recent Role",
            value: "$role at $company",
            category: "Experience & Education",
          ));
        }
        if (duration.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "exp_current_duration",
            label: "Recent Role Duration",
            value: duration,
            category: "Experience & Education",
          ));
        }
      }
    }

    if (_profileData["education"] is List && (_profileData["education"] as List).isNotEmpty) {
      final firstEducation = (_profileData["education"] as List).first;
      if (firstEducation is Map) {
        final degree = firstEducation["degree"]?.toString().trim() ?? "";
        final institution = firstEducation["institution"]?.toString().trim() ?? "";
        final year = firstEducation["year"]?.toString().trim() ?? "";
        final grade = firstEducation["grade"]?.toString().trim() ?? "";
        if (degree.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "edu_degree",
            label: "Highest Degree",
            value: degree,
            category: "Experience & Education",
          ));
        }
        if (institution.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "edu_institution",
            label: "University / Institution",
            value: institution,
            category: "Experience & Education",
          ));
        }
        if (year.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "edu_year",
            label: "Graduation Year",
            value: year,
            category: "Experience & Education",
          ));
        }
        if (grade.isNotEmpty) {
          baseItems.add(QuickFillItem(
            id: "edu_gpa_grade",
            label: "GPA / Grade",
            value: grade,
            category: "Experience & Education",
          ));
        }
      }
    }

    if (_profileData["skills"] is List && (_profileData["skills"] as List).isNotEmpty) {
      final skillsList = (_profileData["skills"] as List).map((entry) => entry.toString()).toList();
      baseItems.add(QuickFillItem(
        id: "skills_summary",
        label: "Technical Skills",
        value: skillsList.join(", "),
        category: "Experience & Education",
        isMultiLine: true,
      ));
    }

    final bioSummary = (_profileData["bio_experience_text"] ?? _profileData["bio_summary"] ?? "").toString().trim();
    if (bioSummary.isNotEmpty && bioSummary != "bio") {
      baseItems.add(QuickFillItem(
        id: "bio_summary",
        label: "Bio / Elevator Pitch",
        value: bioSummary,
        category: "Experience & Education",
        isMultiLine: true,
      ));
    }

    final workAuthVal = _customFormAnswers["work_authorization"]?.toString() ?? "Legally authorized to work without sponsorship";
    baseItems.add(QuickFillItem(
      id: "qa_work_authorization",
      label: "Work Authorization",
      value: workAuthVal,
      category: "Work Authorization",
    ));

    final visaSponsorshipVal = _customFormAnswers["visa_sponsorship"]?.toString() ?? "No, I do not require visa sponsorship";
    baseItems.add(QuickFillItem(
      id: "qa_visa_sponsorship",
      label: "Visa Sponsorship Requirement",
      value: visaSponsorshipVal,
      category: "Work Authorization",
    ));

    final noticePeriodVal = _customFormAnswers["notice_period"]?.toString() ?? "Immediate (available within 1-2 weeks)";
    baseItems.add(QuickFillItem(
      id: "qa_notice_period",
      label: "Notice Period / Start Date",
      value: noticePeriodVal,
      category: "Work Authorization",
    ));

    final expectedSalaryVal = _customFormAnswers["expected_salary"]?.toString() ?? "";
    if (expectedSalaryVal.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "qa_expected_salary",
        label: "Expected Salary",
        value: expectedSalaryVal,
        category: "Work Authorization",
      ));
    }

    final currentSalaryVal = _customFormAnswers["current_salary"]?.toString() ?? "";
    if (currentSalaryVal.isNotEmpty) {
      baseItems.add(QuickFillItem(
        id: "qa_current_salary",
        label: "Current Salary",
        value: currentSalaryVal,
        category: "Work Authorization",
      ));
    }

    final relocationVal = _customFormAnswers["willing_to_relocate"]?.toString() ?? "Yes, open to relocation";
    baseItems.add(QuickFillItem(
      id: "qa_willing_to_relocate",
      label: "Willingness to Relocate",
      value: relocationVal,
      category: "Work Authorization",
    ));

    for (final customEntry in _customFields) {
      final id = customEntry["id"]?.toString() ?? UniqueKey().toString();
      final label = customEntry["label"]?.toString() ?? "";
      final value = customEntry["value"]?.toString() ?? "";
      final category = customEntry["category"]?.toString() ?? "Custom Answers";

      if (label.isNotEmpty && value.isNotEmpty) {
        baseItems.add(QuickFillItem(
          id: id,
          label: label,
          value: value,
          category: category,
          isCustom: true,
          isMultiLine: value.contains("\n") || value.length > 80,
        ));
      }
    }

    final List<QuickFillItem> finalItems = [];
    for (final item in baseItems) {
      if (_deletedFieldIds.contains(item.id)) {
        continue;
      }

      if (_fieldOverrides.containsKey(item.id)) {
        final overrideMap = _fieldOverrides[item.id];
        if (overrideMap is Map) {
          final overriddenLabel = overrideMap["label"]?.toString().trim();
          final overriddenValue = overrideMap["value"]?.toString().trim();
          final overriddenCategory = overrideMap["category"]?.toString().trim();

          finalItems.add(item.copyWith(
            label: (overriddenLabel != null && overriddenLabel.isNotEmpty) ? overriddenLabel : item.label,
            value: (overriddenValue != null && overriddenValue.isNotEmpty) ? overriddenValue : item.value,
            category: (overriddenCategory != null && overriddenCategory.isNotEmpty) ? overriddenCategory : item.category,
          ));
          continue;
        }
      }

      finalItems.add(item);
    }

    return finalItems;
  }

  List<QuickFillItem> _getFilteredItems() {
    final allItems = _buildAllItems();
    final searchQuery = _searchController.text.trim().toLowerCase();

    return allItems.where((item) {
      final matchesCategory = _selectedCategory == "All" || item.category == _selectedCategory;
      if (!matchesCategory) return false;

      if (searchQuery.isEmpty) return true;
      final matchesLabel = item.label.toLowerCase().contains(searchQuery);
      final matchesValue = item.value.toLowerCase().contains(searchQuery);
      return matchesLabel || matchesValue;
    }).toList();
  }

  Future<void> _copyToClipboard(QuickFillItem item) async {
    await Clipboard.setData(ClipboardData(text: item.value));

    _copiedResetTimer?.cancel();
    setState(() {
      _copiedItemId = item.id;
    });

    _copiedResetTimer = Timer(const Duration(milliseconds: 1800), () {
      if (mounted) {
        setState(() {
          _copiedItemId = "";
        });
      }
    });

    if (!mounted) return;
    ScaffoldMessenger.of(context).hideCurrentSnackBar();
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Row(
          children: [
            const Icon(Icons.check_circle_outline, color: Colors.white, size: 18),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                'Copied "${item.label}" to clipboard',
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ],
        ),
        duration: const Duration(milliseconds: 1600),
        behavior: SnackBarBehavior.floating,
        backgroundColor: AppColors.slate900,
      ),
    );
  }

  Future<void> _showAddOrEditFieldDialog({QuickFillItem? existingItem}) async {
    final labelController = TextEditingController(text: existingItem?.label ?? "");
    final valueController = TextEditingController(text: existingItem?.value ?? "");
    String selectedDialogCategory = existingItem?.category ?? "Custom Answers";

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setModalState) {
          return AlertDialog(
            title: Text(existingItem != null ? "Edit ${existingItem.label}" : "Add Custom Field"),
            content: SingleChildScrollView(
              child: SizedBox(
                width: 440,
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    TextField(
                      controller: labelController,
                      decoration: const InputDecoration(
                        labelText: "Field Label *",
                        hintText: "e.g. Notice Period, US Citizen, Why Us Blurb",
                      ),
                    ),
                    const SizedBox(height: 12),
                    DropdownButtonFormField<String>(
                      initialValue: selectedDialogCategory,
                      decoration: const InputDecoration(
                        labelText: "Category",
                      ),
                      items: const [
                        DropdownMenuItem(value: "Work Authorization", child: Text("Work Authorization")),
                        DropdownMenuItem(value: "Personal & Contact", child: Text("Personal & Contact")),
                        DropdownMenuItem(value: "Socials & Links", child: Text("Socials & Links")),
                        DropdownMenuItem(value: "Experience & Education", child: Text("Experience & Education")),
                        DropdownMenuItem(value: "Custom Answers", child: Text("Custom Answers")),
                      ],
                      onChanged: (newVal) {
                        if (newVal != null) {
                          setModalState(() {
                            selectedDialogCategory = newVal;
                          });
                        }
                      },
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: valueController,
                      maxLines: 4,
                      decoration: const InputDecoration(
                        labelText: "Value to Copy *",
                        hintText: "Enter the exact answer or content to paste...",
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
                  if (labelController.text.trim().isEmpty || valueController.text.trim().isEmpty) {
                    return;
                  }
                  Navigator.of(dialogContext).pop(true);
                },
                child: const Text("Save"),
              ),
            ],
          );
        },
      ),
    );

    if (result == true && mounted) {
      final newLabel = labelController.text.trim();
      final newValue = valueController.text.trim();

      setState(() {
        if (existingItem == null) {
          final newCustomField = <String, dynamic>{
            "id": "custom_${DateTime.now().millisecondsSinceEpoch}",
            "label": newLabel,
            "value": newValue,
            "category": selectedDialogCategory,
          };
          _customFields.add(newCustomField);
        } else if (existingItem.isCustom) {
          final existingIndex = _customFields.indexWhere((entry) => entry["id"]?.toString() == existingItem.id);
          final updatedCustomField = <String, dynamic>{
            "id": existingItem.id,
            "label": newLabel,
            "value": newValue,
            "category": selectedDialogCategory,
          };
          if (existingIndex >= 0) {
            _customFields[existingIndex] = updatedCustomField;
          } else {
            _customFields.add(updatedCustomField);
          }
        } else {
          _fieldOverrides[existingItem.id] = {
            "label": newLabel,
            "value": newValue,
            "category": selectedDialogCategory,
          };
          if (existingItem.id.startsWith("qa_")) {
            final questionKey = existingItem.id.replaceFirst("qa_", "");
            _customFormAnswers[questionKey] = newValue;
          }
        }
      });

      await _persistCustomFormAnswers();
    }

    labelController.dispose();
    valueController.dispose();
  }

  Future<void> _confirmDeleteField(QuickFillItem item) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text("Delete Field"),
        content: Text(
          "Are you sure you want to remove \"${item.label}\" from Quick Fill?\n\nThis will only remove it from your copy vault without affecting your underlying profile records.",
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            style: ElevatedButton.styleFrom(
              backgroundColor: AppColors.error,
              foregroundColor: Colors.white,
            ),
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: const Text("Delete"),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      setState(() {
        if (item.isCustom) {
          _customFields.removeWhere((entry) => entry["id"]?.toString() == item.id);
        } else {
          _deletedFieldIds.add(item.id);
          _fieldOverrides.remove(item.id);
        }
      });

      ScaffoldMessenger.of(context).hideCurrentSnackBar();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text('Removed "${item.label}" from Quick Fill'),
          duration: const Duration(seconds: 4),
          behavior: SnackBarBehavior.floating,
          action: SnackBarAction(
            label: "Undo",
            onPressed: () async {
              setState(() {
                _deletedFieldIds.remove(item.id);
              });
              await _persistCustomFormAnswers();
            },
          ),
        ),
      );

      await _persistCustomFormAnswers();
    }
  }

  Future<void> _showRestoreDefaultsDialog() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: const Text("Restore Default Fields"),
        content: const Text(
          "This will restore all deleted pre-filled fields and reset field overrides back to their original values. Custom fields you created will not be touched.",
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            onPressed: () => Navigator.of(dialogContext).pop(true),
            child: const Text("Restore"),
          ),
        ],
      ),
    );

    if (confirmed == true && mounted) {
      setState(() {
        _deletedFieldIds.clear();
        _fieldOverrides.clear();
      });

      await _persistCustomFormAnswers();

      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(
          content: Text("Default fields restored"),
          duration: Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    }
  }

  Future<void> _showViewFullContentDialog(QuickFillItem item) async {
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Row(
          children: [
            Icon(
              _getCategoryIcon(item.category),
              size: 20,
              color: AppColors.primary,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                item.label,
                style: const TextStyle(
                  fontSize: 16,
                  fontWeight: FontWeight.w700,
                ),
                overflow: TextOverflow.ellipsis,
              ),
            ),
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
              decoration: BoxDecoration(
                color: AppColors.surfaceContainerHigh,
                borderRadius: BorderRadius.circular(12),
              ),
              child: Text(
                item.category,
                style: const TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
            ),
          ],
        ),
        content: SizedBox(
          width: 520,
          child: Container(
            constraints: const BoxConstraints(maxHeight: 380),
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppColors.surfaceContainerLow,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: AppColors.outlineVariant),
            ),
            child: SingleChildScrollView(
              child: SelectableText(
                item.value,
                style: const TextStyle(
                  fontSize: 13,
                  fontFamily: "monospace",
                  height: 1.5,
                  color: AppColors.onSurface,
                ),
              ),
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(),
            child: const Text("Close"),
          ),
          OutlinedButton.icon(
            onPressed: () {
              Navigator.of(dialogContext).pop();
              _showAddOrEditFieldDialog(existingItem: item);
            },
            icon: const Icon(Icons.edit_outlined, size: 16),
            label: const Text("Edit"),
          ),
          FilledButton.icon(
            onPressed: () {
              _copyToClipboard(item);
              Navigator.of(dialogContext).pop();
            },
            icon: const Icon(Icons.copy, size: 16),
            label: const Text("Copy Value"),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final isWide = MediaQuery.of(context).size.width >= 960;

    return Scaffold(
      backgroundColor: AppColors.background,
      body: _isLoading
          ? const Center(child: CircularProgressIndicator())
          : RefreshIndicator(
              onRefresh: _loadPreferences,
              child: SingleChildScrollView(
                physics: const AlwaysScrollableScrollPhysics(),
                padding: EdgeInsets.symmetric(
                  horizontal: isWide ? 32 : 16,
                  vertical: 24,
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    _buildHeader(isWide),
                    const SizedBox(height: 16),
                    _buildSearchBar(),
                    const SizedBox(height: 12),
                    _buildCategoryFilterBar(),
                    const SizedBox(height: 20),
                    _buildItemsContent(isWide),
                  ],
                ),
              ),
            ),
      floatingActionButton: isWide
          ? null
          : FloatingActionButton.extended(
              onPressed: () => _showAddOrEditFieldDialog(),
              backgroundColor: AppColors.primary,
              foregroundColor: Colors.white,
              icon: const Icon(Icons.add),
              label: const Text("Add Field"),
            ),
    );
  }

  Widget _buildHeader(bool isWide) {
    final hasModifications = _deletedFieldIds.isNotEmpty || _fieldOverrides.isNotEmpty;

    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Container(
          width: 40,
          height: 40,
          decoration: BoxDecoration(
            color: AppColors.primary.withValues(alpha: 0.1),
            borderRadius: BorderRadius.circular(10),
          ),
          child: const Icon(
            Icons.bolt,
            color: AppColors.primary,
            size: 24,
          ),
        ),
        const SizedBox(width: 14),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: const [
              Text(
                "Quick Fill Vault",
                style: TextStyle(
                  fontSize: 24,
                  fontWeight: FontWeight.w800,
                  color: AppColors.onSurface,
                ),
              ),
              SizedBox(height: 4),
              Text(
                "One-tap copy for all your application answers, links, and custom form details.",
                style: TextStyle(
                  fontSize: 13,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
            ],
          ),
        ),
        if (isWide) ...[
          if (hasModifications) ...[
            TextButton.icon(
              onPressed: _showRestoreDefaultsDialog,
              icon: const Icon(Icons.restore, size: 16),
              label: const Text("Restore Defaults"),
            ),
            const SizedBox(width: 8),
          ],
          OutlinedButton.icon(
            onPressed: () => _showAddOrEditFieldDialog(),
            icon: const Icon(Icons.add_circle_outline, size: 18),
            label: const Text("Add Custom Field"),
          ),
        ],
      ],
    );
  }

  Widget _buildSearchBar() {
    return TextField(
      controller: _searchController,
      decoration: InputDecoration(
        hintText: "Search any field name or value (e.g. visa, salary, email)...",
        prefixIcon: const Icon(Icons.search, size: 20, color: AppColors.outline),
        suffixIcon: _searchController.text.isNotEmpty
            ? IconButton(
                icon: const Icon(Icons.clear, size: 18),
                onPressed: () {
                  _searchController.clear();
                },
              )
            : null,
        filled: true,
        fillColor: AppColors.surfaceContainerLowest,
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        border: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: AppColors.outlineVariant),
        ),
        enabledBorder: OutlineInputBorder(
          borderRadius: BorderRadius.circular(10),
          borderSide: const BorderSide(color: AppColors.outlineVariant),
        ),
      ),
    );
  }

  Widget _buildCategoryFilterBar() {
    final hasModifications = _deletedFieldIds.isNotEmpty || _fieldOverrides.isNotEmpty;

    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: [
          ..._categoryFilterOptions.map((category) {
            final isSelected = _selectedCategory == category;
            return Padding(
              padding: const EdgeInsets.only(right: 8.0),
              child: ChoiceChip(
                label: Text(category),
                selected: isSelected,
                onSelected: (selected) {
                  if (selected) {
                    setState(() {
                      _selectedCategory = category;
                    });
                  }
                },
                labelStyle: TextStyle(
                  fontSize: 12,
                  fontWeight: isSelected ? FontWeight.w700 : FontWeight.w500,
                  color: isSelected ? Colors.white : AppColors.onSurfaceVariant,
                ),
                selectedColor: AppColors.primary,
                backgroundColor: AppColors.surfaceContainerLowest,
                shape: RoundedRectangleBorder(
                  borderRadius: BorderRadius.circular(20),
                  side: BorderSide(
                    color: isSelected ? AppColors.primary : AppColors.outlineVariant,
                  ),
                ),
                showCheckmark: false,
              ),
            );
          }),
          if (!MediaQuery.of(context).size.width.isNegative && MediaQuery.of(context).size.width < 960 && hasModifications)
            Padding(
              padding: const EdgeInsets.only(left: 4.0),
              child: ActionChip(
                avatar: const Icon(Icons.restore, size: 14),
                label: const Text("Restore Defaults", style: TextStyle(fontSize: 12)),
                onPressed: _showRestoreDefaultsDialog,
              ),
            ),
        ],
      ),
    );
  }

  Widget _buildItemsContent(bool isWide) {
    final items = _getFilteredItems();

    if (items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 48.0),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.content_paste_off, size: 48, color: AppColors.outline),
              const SizedBox(height: 12),
              Text(
                _searchController.text.isNotEmpty
                    ? 'No fields matching "${_searchController.text}"'
                    : "No fields available in this category",
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 6),
              const Text(
                "Tap 'Add Field' to create a custom answer for forms.",
                style: TextStyle(fontSize: 13, color: AppColors.outline),
              ),
            ],
          ),
        ),
      );
    }

    if (isWide) {
      return LayoutBuilder(
        builder: (context, constraints) {
          final columnCount = constraints.maxWidth >= 1400 ? 3 : 2;
          final itemWidth = (constraints.maxWidth - (columnCount - 1) * 16) / columnCount;

          return Wrap(
            spacing: 16,
            runSpacing: 16,
            children: items.map((item) {
              return SizedBox(
                width: itemWidth,
                child: _buildItemCard(item),
              );
            }).toList(),
          );
        },
      );
    }

    return ListView.separated(
      shrinkWrap: true,
      physics: const NeverScrollableScrollPhysics(),
      itemCount: items.length,
      separatorBuilder: (_, _) => const SizedBox(height: 10),
      itemBuilder: (context, index) => _buildItemCard(items[index]),
    );
  }

  Widget _buildItemCard(QuickFillItem item) {
    final isCopied = _copiedItemId == item.id;
    final hasMoreContent = item.value.length > 70 || item.value.contains("\n");

    return Container(
      decoration: BoxDecoration(
        color: AppColors.surfaceContainerLowest,
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isCopied ? AppColors.successGreen : AppColors.outlineVariant,
          width: isCopied ? 1.5 : 1.0,
        ),
        boxShadow: [
          BoxShadow(
            color: Colors.black.withValues(alpha: 0.02),
            blurRadius: 6,
            offset: const Offset(0, 2),
          ),
        ],
      ),
      child: Material(
        color: Colors.transparent,
        borderRadius: BorderRadius.circular(12),
        child: InkWell(
          borderRadius: BorderRadius.circular(12),
          onTap: () => _copyToClipboard(item),
          child: Padding(
            padding: const EdgeInsets.all(14.0),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              mainAxisSize: MainAxisSize.min,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Expanded(
                      child: Row(
                        children: [
                          Icon(
                            _getCategoryIcon(item.category),
                            size: 16,
                            color: AppColors.onSurfaceVariant,
                          ),
                          const SizedBox(width: 6),
                          Expanded(
                            child: Text(
                              item.label,
                              style: const TextStyle(
                                fontSize: 13,
                                fontWeight: FontWeight.w700,
                                color: AppColors.onSurface,
                              ),
                              overflow: TextOverflow.ellipsis,
                            ),
                          ),
                        ],
                      ),
                    ),
                    Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          icon: const Icon(Icons.edit_outlined, size: 16, color: AppColors.onSurfaceVariant),
                          tooltip: "Edit field",
                          padding: EdgeInsets.zero,
                          constraints: const BoxConstraints(),
                          onPressed: () => _showAddOrEditFieldDialog(existingItem: item),
                        ),
                        const SizedBox(width: 8),
                        IconButton(
                          icon: const Icon(Icons.delete_outline, size: 16, color: AppColors.error),
                          tooltip: "Delete field",
                          padding: EdgeInsets.zero,
                          constraints: const BoxConstraints(),
                          onPressed: () => _confirmDeleteField(item),
                        ),
                        const SizedBox(width: 8),
                        AnimatedContainer(
                          duration: const Duration(milliseconds: 200),
                          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
                          decoration: BoxDecoration(
                            color: isCopied ? AppColors.successGreen : AppColors.surfaceContainerHigh,
                            borderRadius: BorderRadius.circular(6),
                          ),
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              Icon(
                                isCopied ? Icons.check : Icons.copy,
                                size: 13,
                                color: isCopied ? Colors.white : AppColors.onSurfaceVariant,
                              ),
                              const SizedBox(width: 4),
                              Text(
                                isCopied ? "Copied" : "Copy",
                                style: TextStyle(
                                  fontSize: 11,
                                  fontWeight: FontWeight.w700,
                                  color: isCopied ? Colors.white : AppColors.onSurfaceVariant,
                                ),
                              ),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
                const SizedBox(height: 10),
                Container(
                  width: double.infinity,
                  padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                  decoration: BoxDecoration(
                    color: AppColors.surfaceContainerLow,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: AppColors.outlineVariant.withValues(alpha: 0.6)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        item.value,
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                        style: const TextStyle(
                          fontSize: 13,
                          fontFamily: "monospace",
                          color: AppColors.onSurface,
                        ),
                      ),
                      if (hasMoreContent) ...[
                        const SizedBox(height: 6),
                        Align(
                          alignment: Alignment.centerRight,
                          child: InkWell(
                            onTap: () => _showViewFullContentDialog(item),
                            borderRadius: BorderRadius.circular(4),
                            child: Padding(
                              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
                              child: Row(
                                mainAxisSize: MainAxisSize.min,
                                children: const [
                                  Text(
                                    "More",
                                    style: TextStyle(
                                      fontSize: 11,
                                      fontWeight: FontWeight.w700,
                                      color: AppColors.primary,
                                    ),
                                  ),
                                  SizedBox(width: 2),
                                  Icon(
                                    Icons.unfold_more,
                                    size: 13,
                                    color: AppColors.primary,
                                  ),
                                ],
                              ),
                            ),
                          ),
                        ),
                      ],
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  IconData _getCategoryIcon(String category) {
    switch (category) {
      case "Personal & Contact":
        return Icons.person_outline;
      case "Work Authorization":
        return Icons.verified_user_outlined;
      case "Socials & Links":
        return Icons.link;
      case "Experience & Education":
        return Icons.school_outlined;
      case "Custom Answers":
      default:
        return Icons.tune_outlined;
    }
  }
}
