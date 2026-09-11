/// Represents a form autofill item that can be copied with one tap during job applications.
class QuickFillItem {
  final String id;
  final String label;
  final String value;
  final String category;
  final bool isCustom;
  final bool isMultiLine;

  const QuickFillItem({
    required this.id,
    required this.label,
    required this.value,
    required this.category,
    this.isCustom = false,
    this.isMultiLine = false,
  });

  QuickFillItem copyWith({
    String? id,
    String? label,
    String? value,
    String? category,
    bool? isCustom,
    bool? isMultiLine,
  }) {
    return QuickFillItem(
      id: id ?? this.id,
      label: label ?? this.label,
      value: value ?? this.value,
      category: category ?? this.category,
      isCustom: isCustom ?? this.isCustom,
      isMultiLine: isMultiLine ?? this.isMultiLine,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'label': label,
      'value': value,
      'category': category,
      'is_custom': isCustom,
      'is_multiline': isMultiLine,
    };
  }

  factory QuickFillItem.fromJson(Map<String, dynamic> json) {
    return QuickFillItem(
      id: json['id']?.toString() ?? '',
      label: json['label']?.toString() ?? '',
      value: json['value']?.toString() ?? '',
      category: json['category']?.toString() ?? 'Custom Answers',
      isCustom: json['is_custom'] == true || json['is_custom'] == null,
      isMultiLine: json['is_multiline'] == true,
    );
  }
}
