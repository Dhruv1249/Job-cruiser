/// Model class representing a user's tracked job application in the CRM pipeline.
class JobApplication {
  const JobApplication({
    required this.applicationId,
    required this.jobId,
    required this.status,
    required this.title,
    required this.companyId,
    this.location,
    this.appliedAt,
    this.companyName,
    this.matchScore,
  });

  final String applicationId;
  final String jobId;
  final String status;
  final String title;
  final String companyId;
  final String? location;
  final String? appliedAt;
  final String? companyName;
  final int? matchScore;

  /// Creates a [JobApplication] instance from a JSON map.
  factory JobApplication.fromJson(Map<String, dynamic> json) {
    return JobApplication(
      applicationId: json['application_id'] as String? ?? json['id'] as String? ?? '',
      jobId: json['job_id'] as String? ?? '',
      status: json['status'] as String? ?? 'bookmarked',
      title: json['title'] as String? ?? 'Position',
      companyId: json['company_id'] as String? ?? '',
      location: json['location'] as String?,
      appliedAt: json['applied_at'] as String?,
      companyName: json['company_name'] as String? ?? json['company'] as String?,
      matchScore: (json['match_score'] as num?)?.toInt(),
    );
  }

  /// Helper getter to format user-friendly status labels.
  String get statusDisplayLabel {
    switch (status.toLowerCase()) {
      case 'bookmarked':
        return 'Bookmarked';
      case 'applied':
        return 'Applied';
      case 'outreach_sent':
        return 'Outreach Sent';
      case 'interview':
      case 'interviewing':
        return 'Interview';
      case 'offer':
        return 'Offer';
      case 'rejected':
        return 'Rejected';
      default:
        return status;
    }
  }
}
