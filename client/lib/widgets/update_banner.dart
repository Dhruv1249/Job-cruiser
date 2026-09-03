import 'package:flutter/material.dart';
import 'package:url_launcher/url_launcher.dart';
import '../main.dart' show AppColors;
import '../services/update_checker_service.dart';

/// Persistent top banner that appears when a newer version is available on GitHub Releases.
///
/// Provides a one-tap action to download the APK in-app and trigger the Android
/// system package installer directly.
class UpdateBanner extends StatefulWidget {
  const UpdateBanner({
    super.key,
    required this.update,
    required this.onDismiss,
    this.updateCheckerService,
  });

  final PendingUpdate update;
  final VoidCallback onDismiss;
  final UpdateCheckerService? updateCheckerService;

  @override
  State<UpdateBanner> createState() => _UpdateBannerState();
}

class _UpdateBannerState extends State<UpdateBanner> {
  late final UpdateCheckerService _service =
      widget.updateCheckerService ?? UpdateCheckerService();

  bool _isDownloading = false;
  double _progress = 0.0;

  Future<void> _handleInstall() async {
    if (_isDownloading) return;

    setState(() {
      _isDownloading = true;
      _progress = 0.0;
    });

    final success = await _service.downloadAndInstall(
      update: widget.update,
      onProgress: (progressValue) {
        if (mounted) {
          setState(() {
            _progress = progressValue;
          });
        }
      },
    );

    if (!mounted) return;

    setState(() {
      _isDownloading = false;
      _progress = 0.0;
    });

    if (!success) {
      final uri = Uri.tryParse(widget.update.downloadUrl);
      if (uri != null && await canLaunchUrl(uri)) {
        await launchUrl(uri, mode: LaunchMode.externalApplication);
      } else {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Failed to download update. Please try again.'),
            backgroundColor: AppColors.error,
          ),
        );
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Material(
      color: Colors.transparent,
      child: Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        decoration: BoxDecoration(
          color: AppColors.primary,
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.12),
              blurRadius: 4,
              offset: const Offset(0, 2),
            ),
          ],
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.system_update_alt, color: Colors.white, size: 18),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Text(
                        'Update available — v${widget.update.versionName}',
                        style: const TextStyle(
                          color: Colors.white,
                          fontWeight: FontWeight.w700,
                          fontSize: 13,
                        ),
                      ),
                      if (widget.update.releaseNotes.isNotEmpty) ...[
                        const SizedBox(height: 2),
                        Text(
                          widget.update.releaseNotes,
                          style: const TextStyle(
                            color: Colors.white70,
                            fontSize: 11,
                          ),
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                TextButton(
                  onPressed: _isDownloading ? null : _handleInstall,
                  style: TextButton.styleFrom(
                    backgroundColor: Colors.white,
                    foregroundColor: AppColors.primary,
                    disabledBackgroundColor: Colors.white70,
                    disabledForegroundColor: AppColors.primary,
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                    minimumSize: const Size(0, 32),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(6)),
                  ),
                  child: _isDownloading
                      ? Text(
                          _progress > 0
                              ? '${(_progress * 100).toInt()}%'
                              : 'Downloading...',
                          style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
                        )
                      : const Text(
                          'Update',
                          style: TextStyle(fontSize: 12, fontWeight: FontWeight.w700),
                        ),
                ),
                const SizedBox(width: 4),
                IconButton(
                  icon: const Icon(Icons.close, color: Colors.white70, size: 18),
                  onPressed: _isDownloading ? null : widget.onDismiss,
                  padding: EdgeInsets.zero,
                  constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                  tooltip: 'Dismiss',
                ),
              ],
            ),
            if (_isDownloading) ...[
              const SizedBox(height: 8),
              ClipRRect(
                borderRadius: BorderRadius.circular(2),
                child: LinearProgressIndicator(
                  value: _progress > 0 ? _progress : null,
                  backgroundColor: Colors.white24,
                  valueColor: const AlwaysStoppedAnimation<Color>(Colors.white),
                  minHeight: 3,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }
}
