import 'package:flutter/material.dart';
import '../main.dart' show AppColors;

/// Reusable widget for dynamically displaying company logos using free APIs
/// (Clearbit Logo API & Google Favicon API) with stylized initial letter fallbacks.
class CompanyLogoAvatar extends StatefulWidget {
  const CompanyLogoAvatar({
    super.key,
    required this.companyName,
    this.jobUrl = '',
    this.size = 28.0,
  });

  final String companyName;
  final String jobUrl;
  final double size;

  @override
  State<CompanyLogoAvatar> createState() => _CompanyLogoAvatarState();
}

class _CompanyLogoAvatarState extends State<CompanyLogoAvatar> {
  int _logoStage = 0; // 0: Clearbit, 1: Google Favicon, 2: Letter Fallback

  String? _extractDomain() {
    final cleanUrl = widget.jobUrl.trim();
    if (cleanUrl.isNotEmpty) {
      final uri = Uri.tryParse(cleanUrl);
      if (uri != null && uri.host.isNotEmpty) {
        String host = uri.host.toLowerCase();
        if (host.startsWith('www.')) {
          host = host.substring(4);
        }
        final parts = host.split('.');
        if (parts.length >= 2) {
          final secondLast = parts[parts.length - 2];
          if (secondLast == 'greenhouse' || secondLast == 'lever' || secondLast == 'ashbyhq' || secondLast == 'workdaysite') {
            final pathSegments = uri.pathSegments.where((s) => s.isNotEmpty).toList();
            if (pathSegments.isNotEmpty) {
              return '${pathSegments.first.toLowerCase()}.com';
            }
          }
          return host;
        }
      }
    }

    final cleanCompany = widget.companyName.trim().replaceAll(RegExp(r'[^a-zA-Z0-9]'), '').toLowerCase();
    if (cleanCompany.isNotEmpty && cleanCompany != 'unknown') {
      return '$cleanCompany.com';
    }

    return null;
  }

  @override
  Widget build(BuildContext context) {
    final domain = _extractDomain();

    if (domain == null || _logoStage >= 2) {
      return _buildLetterAvatar();
    }

    final String logoUrl = _logoStage == 0
        ? 'https://logo.clearbit.com/$domain'
        : 'https://www.google.com/s2/favicons?domain=$domain&sz=128';

    return ClipRRect(
      borderRadius: BorderRadius.circular(widget.size * 0.25),
      child: Container(
        width: widget.size,
        height: widget.size,
        color: AppColors.surfaceContainerLowest,
        child: Image.network(
          logoUrl,
          width: widget.size,
          height: widget.size,
          fit: BoxFit.contain,
          errorBuilder: (context, error, stackTrace) {
            WidgetsBinding.instance.addPostFrameCallback((_) {
              if (mounted) {
                setState(() {
                  _logoStage += 1;
                });
              }
            });
            return _buildLetterAvatar();
          },
        ),
      ),
    );
  }

  Widget _buildLetterAvatar() {
    final firstLetter = widget.companyName.trim().isNotEmpty
        ? widget.companyName.trim()[0].toUpperCase()
        : 'C';

    return Container(
      width: widget.size,
      height: widget.size,
      decoration: BoxDecoration(
        color: AppColors.primaryContainer,
        borderRadius: BorderRadius.circular(widget.size * 0.25),
        border: Border.all(color: AppColors.outlineVariant, width: 1.0),
      ),
      child: Center(
        child: Text(
          firstLetter,
          style: TextStyle(
            color: AppColors.onPrimaryContainer,
            fontSize: widget.size * 0.45,
            fontWeight: FontWeight.w700,
          ),
        ),
      ),
    );
  }
}
