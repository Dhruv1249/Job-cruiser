import "dart:async";
import "package:flutter/material.dart";
import "package:flutter/services.dart";
import "../main.dart" show AppColors;
import "../models/quick_fill_item.dart";
import "../services/api_service.dart";

/// Screen providing single-tap clipboard copy targets for job application forms and custom user answers.
class QuickFillScreen extends StatefulWidget {
  final Map<String, dynamic>? initialProfileData;

  const QuickFillScreen({
    super.key,
    this.initialProfileData,
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
      _customFormAnswers = rawCustomAnswers;
      _customFields = rawCustomFields;
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

      setState(() {
        _profileData = data;
        _customFormAnswers = rawCustomAnswers;
        _customFields = rawCustomFields;
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

    final Map<String, dynamic> payload = Map<String, dynamic>.from(_profileData);
    payload["custom_form_answers"] = updatedPayload;

    await _apiService.savePreferences(payload);
  }

  List<QuickFillItem> _buildAllItems() {
    final List<QuickFillItem> items = [];

    final fullName = (_profileData["full_name"] ?? "").toString().trim();
    if (fullName.isNotEmpty && fullName != "User") {
      items.add(QuickFillItem(
        id: "profile_full_name",
        label: "Full Name",
        value: fullName,
        category: "Personal & Contact",
      ));

      final nameTokens = fullName.split(" ").where((token) => token.trim().isNotEmpty).toList();
      if (nameTokens.isNotEmpty) {
        items.add(QuickFillItem(
          id: "profile_first_name",
          label: "First Name",
          value: nameTokens.first,
          category: "Personal & Contact",
        ));
      }
      if (nameTokens.length > 1) {
        items.add(QuickFillItem(
          id: "profile_last_name",
          label: "Last Name",
          value: nameTokens.sublist(1).join(" "),
          category: "Personal & Contact",
        ));
      }
    }

    final email = (_profileData["email"] ?? "").toString().trim();
    if (email.isNotEmpty) {
      items.add(QuickFillItem(
        id: "profile_email",
        label: "Email Address",
        value: email,
        category: "Personal & Contact",
      ));
    }

    final phone = (_profileData["phone"] ?? "").toString().trim();
    if (phone.isNotEmpty) {
      items.add(QuickFillItem(
        id: "profile_phone",
        label: "Phone Number",
        value: phone,
        category: "Personal & Contact",
      ));
    }

    final location = (_profileData["location"] ?? "").toString().trim();
    if (location.isNotEmpty) {
      items.add(QuickFillItem(
        id: "profile_location",
        label: "Current Location",
        value: location,
        category: "Personal & Contact",
      ));
    }

    final country = (_profileData["country"] ?? "").toString().trim();
    if (country.isNotEmpty) {
      items.add(QuickFillItem(
        id: "profile_country",
        label: "Country",
        value: country,
        category: "Personal & Contact",
      ));
    }

    final linkedIn = (_profileData["linkedin_url"] ?? "").toString().trim();
    if (linkedIn.isNotEmpty) {
      items.add(QuickFillItem(
        id: "link_linkedin",
        label: "LinkedIn Profile URL",
        value: linkedIn,
        category: "Socials & Links",
      ));
    }

    final github = (_profileData["github_url"] ?? "").toString().trim();
    if (github.isNotEmpty) {
      items.add(QuickFillItem(
        id: "link_github",
        label: "GitHub Profile URL",
        value: github,
        category: "Socials & Links",
      ));
    }

    final portfolio = (_profileData["portfolio_url"] ?? "").toString().trim();
    if (portfolio.isNotEmpty) {
      items.add(QuickFillItem(
        id: "link_portfolio",
        label: "Portfolio / Website URL",
        value: portfolio,
        category: "Socials & Links",
      ));
    }

    if (_profileData["custom_links"] is List) {
      final customLinks = _profileData["custom_links"] as List;
      for (int i = 0; i < customLinks.length; i++) {
        final linkMap = customLinks[i];
        if (linkMap is Map) {
          final label = linkMap["label"]?.toString().trim() ?? "Custom Link";
          final url = linkMap["url"]?.toString().trim() ?? "";
          if (url.isNotEmpty) {
            items.add(QuickFillItem(
              id: "link_custom_$i",
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
          items.add(QuickFillItem(
            id: "exp_current_role",
            label: "Current / Most Recent Role",
            value: "$role at $company",
            category: "Experience & Education",
          ));
        }
        if (duration.isNotEmpty) {
          items.add(QuickFillItem(
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
          items.add(QuickFillItem(
            id: "edu_degree",
            label: "Degree & Major",
            value: degree,
            category: "Experience & Education",
          ));
        }
        if (institution.isNotEmpty) {
          items.add(QuickFillItem(
            id: "edu_institution",
            label: "University / Institution",
            value: institution,
            category: "Experience & Education",
          ));
        }
        if (year.isNotEmpty) {
          items.add(QuickFillItem(
            id: "edu_graduation_year",
            label: "Graduation Year / Period",
            value: year,
            category: "Experience & Education",
          ));
        }
        if (grade.isNotEmpty) {
          items.add(QuickFillItem(
            id: "edu_gpa_grade",
            label: "GPA / Grade",
            value: grade,
            category: "Experience & Education",
          ));
        }
      }
    }

    if (_profileData["skills"] is List && (_profileData["skills"] as List).isNotEmpty) {
      final skillsList = (_profileData["skills"] as List).map((sk) => sk.toString()).toList();
      items.add(QuickFillItem(
        id: "skills_summary",
        label: "Technical Skills",
        value: skillsList.join(", "),
        category: "Experience & Education",
        isMultiLine: true,
      ));
    }

    final bioSummary = (_profileData["bio_experience_text"] ?? _profileData["bio_summary"] ?? "").toString().trim();
    if (bioSummary.isNotEmpty && bioSummary != "bio") {
      items.add(QuickFillItem(
        id: "bio_summary",
        label: "Bio / Elevator Pitch",
        value: bioSummary,
        category: "Experience & Education",
        isMultiLine: true,
      ));
    }

    final workAuthVal = _customFormAnswers["work_authorization"]?.toString() ?? "Legally authorized to work without sponsorship";
    items.add(QuickFillItem(
      id: "qa_work_authorization",
      label: "Work Authorization",
      value: workAuthVal,
      category: "Work Authorization",
    ));

    final visaSponsorshipVal = _customFormAnswers["visa_sponsorship"]?.toString() ?? "No, I do not require visa sponsorship";
    items.add(QuickFillItem(
      id: "qa_visa_sponsorship",
      label: "Visa Sponsorship Requirement",
      value: visaSponsorshipVal,
      category: "Work Authorization",
    ));

    final noticePeriodVal = _customFormAnswers["notice_period"]?.toString() ?? "Immediate (available within 1-2 weeks)";
    items.add(QuickFillItem(
      id: "qa_notice_period",
      label: "Notice Period / Start Date",
      value: noticePeriodVal,
      category: "Work Authorization",
    ));

    final expectedSalaryVal = _customFormAnswers["expected_salary"]?.toString() ?? "";
    if (expectedSalaryVal.isNotEmpty) {
      items.add(QuickFillItem(
        id: "qa_expected_salary",
        label: "Expected Salary",
        value: expectedSalaryVal,
        category: "Work Authorization",
      ));
    }

    final currentSalaryVal = _customFormAnswers["current_salary"]?.toString() ?? "";
    if (currentSalaryVal.isNotEmpty) {
      items.add(QuickFillItem(
        id: "qa_current_salary",
        label: "Current Salary",
        value: currentSalaryVal,
        category: "Work Authorization",
      ));
    }

    final relocationVal = _customFormAnswers["willing_to_relocate"]?.toString() ?? "Yes, open to relocation";
    items.add(QuickFillItem(
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
        items.add(QuickFillItem(
          id: id,
          label: label,
          value: value,
          category: category,
          isCustom: true,
          isMultiLine: value.contains("\n") || value.length > 80,
        ));
      }
    }

    return items;
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

  Future<void> _showAddOrEditCustomFieldDialog({Map<String, dynamic>? existingField, int? editIndex}) async {
    final labelController = TextEditingController(text: existingField?["label"]?.toString() ?? "");
    final valueController = TextEditingController(text: existingField?["value"]?.toString() ?? "");
    String selectedDialogCategory = existingField?["category"]?.toString() ?? "Custom Answers";

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => StatefulBuilder(
        builder: (dialogContext, setModalState) {
          return AlertDialog(
            title: Text(existingField != null ? "Edit Field" : "Add Custom Field"),
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
      final updatedMap = <String, dynamic>{
        "id": existingField?["id"]?.toString() ?? "custom_${DateTime.now().millisecondsSinceEpoch}",
        "label": labelController.text.trim(),
        "value": valueController.text.trim(),
        "category": selectedDialogCategory,
      };

      setState(() {
        if (editIndex != null && editIndex >= 0 && editIndex < _customFields.length) {
          _customFields[editIndex] = updatedMap;
        } else {
          _customFields.add(updatedMap);
        }
      });

      await _persistCustomFormAnswers();
    }

    labelController.dispose();
    valueController.dispose();
  }

  Future<void> _showEditStandardQuestionDialog(QuickFillItem item) async {
    final valueController = TextEditingController(text: item.value);

    final result = await showDialog<bool>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Text("Edit ${item.label}"),
        content: SizedBox(
          width: 440,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: valueController,
                maxLines: 3,
                decoration: InputDecoration(
                  labelText: item.label,
                  hintText: "Enter your preferred response...",
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogContext).pop(false),
            child: const Text("Cancel"),
          ),
          ElevatedButton(
            onPressed: () {
              if (valueController.text.trim().isEmpty) return;
              Navigator.of(dialogContext).pop(true);
            },
            child: const Text("Save"),
          ),
        ],
      ),
    );

    if (result == true && mounted) {
      final questionKey = item.id.replaceFirst("qa_", "");
      setState(() {
        _customFormAnswers[questionKey] = valueController.text.trim();
      });

      await _persistCustomFormAnswers();
    }

    valueController.dispose();
  }

  Future<void> _deleteCustomField(String fieldId) async {
    setState(() {
      _customFields.removeWhere((entry) => entry["id"]?.toString() == fieldId);
    });
    await _persistCustomFormAnswers();

    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text("Custom field removed"),
        duration: Duration(seconds: 1),
        behavior: SnackBarBehavior.floating,
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
                    _buildItemsListOrGrid(isWide),
                  ],
                ),
              ),
            ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _showAddOrEditCustomFieldDialog(),
        backgroundColor: AppColors.primary,
        foregroundColor: Colors.white,
        icon: const Icon(Icons.add),
        label: const Text("Add Field"),
      ),
    );
  }

  Widget _buildHeader(bool isWide) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: [
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
        if (isWide)
          OutlinedButton.icon(
            onPressed: () => _showAddOrEditCustomFieldDialog(),
            icon: const Icon(Icons.add_circle_outline, size: 18),
            label: const Text("Add Custom Field"),
          ),
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
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: Row(
        children: _categoryFilterOptions.map((category) {
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
        }).toList(),
      ),
    );
  }

  Widget _buildItemsListOrGrid(bool isWide) {
    final items = _getFilteredItems();

    if (items.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 48.0),
          child: Column(
            children: [
              const Icon(Icons.content_paste_off, size: 48, color: AppColors.outline),
              const SizedBox(height: 12),
              Text(
                _searchController.text.isNotEmpty
                    ? 'No fields match "${_searchController.text}"'
                    : "No fields available in this category.",
                style: const TextStyle(
                  fontSize: 15,
                  fontWeight: FontWeight.w600,
                  color: AppColors.onSurfaceVariant,
                ),
              ),
              const SizedBox(height: 8),
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
    final isStandardQuestion = item.id.startsWith("qa_");

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
                        if (item.isCustom) ...[
                          IconButton(
                            icon: const Icon(Icons.edit_outlined, size: 16, color: AppColors.onSurfaceVariant),
                            tooltip: "Edit field",
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                            onPressed: () {
                              final editIndex = _customFields.indexWhere((entry) => entry["id"]?.toString() == item.id);
                              final existingMap = editIndex >= 0 ? _customFields[editIndex] : null;
                              _showAddOrEditCustomFieldDialog(
                                existingField: existingMap,
                                editIndex: editIndex >= 0 ? editIndex : null,
                              );
                            },
                          ),
                          const SizedBox(width: 8),
                          IconButton(
                            icon: const Icon(Icons.delete_outline, size: 16, color: AppColors.error),
                            tooltip: "Delete field",
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                            onPressed: () => _deleteCustomField(item.id),
                          ),
                          const SizedBox(width: 8),
                        ] else if (isStandardQuestion) ...[
                          IconButton(
                            icon: const Icon(Icons.edit_outlined, size: 16, color: AppColors.onSurfaceVariant),
                            tooltip: "Edit answer",
                            padding: EdgeInsets.zero,
                            constraints: const BoxConstraints(),
                            onPressed: () => _showEditStandardQuestionDialog(item),
                          ),
                          const SizedBox(width: 8),
                        ],
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
                  child: Text(
                    item.value,
                    maxLines: item.isMultiLine ? 4 : 2,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 13,
                      fontFamily: "monospace",
                      color: AppColors.onSurface,
                    ),
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
        return Icons.assignment_outlined;
    }
  }
}
