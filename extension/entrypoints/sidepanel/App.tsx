import { JSX, render } from 'preact';
import { useCallback, useEffect, useState } from 'preact/hooks';
import browser from 'webextension-polyfill';
import {
  DEFAULT_BACKEND_URL,
  DEFAULT_CONFIGURATION,
  DEFAULT_PROFILE,
  DEFAULT_WEB_PORTAL_URL,
  resolveBackendUrl,
} from '../../lib/storage-service';
import {
  CustomFormField,
  ExtensionConfig,
  ExtensionMessage,
  QuickFillProfile,
} from '../../lib/types';

type ActiveTab = 'vault' | 'skills' | 'settings';

interface NotificationState {
  message: string;
  type: 'success' | 'error';
}

/**
 * Job Cruiser Brand & Action SVG Icons
 */
function IconBolt({ size = 16 }: { size?: number }): JSX.Element {
  return (
    <div style={boltContainerStyle}>
      <svg width={size} height={size} viewBox="0 0 24 24" fill="#000000" stroke="none">
        <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2" />
      </svg>
    </div>
  );
}

function IconRotateCw({ size = 14, isSpinning = false }: { size?: number; isSpinning?: boolean }): JSX.Element {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      style={{
        animation: isSpinning ? 'jc-spin 0.8s linear infinite' : 'none',
        display: 'inline-block',
      }}
    >
      <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
      <path d="M21 3v5h-5" />
    </svg>
  );
}

function IconCopy({ size = 13 }: { size?: number }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect width="14" height="14" x="8" y="8" rx="2" ry="2" />
      <path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" />
    </svg>
  );
}

function IconCheck({ size = 13, color = '#10B981' }: { size?: number; color?: string }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color} strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  );
}



function IconTrash({ size = 13 }: { size?: number }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18" />
      <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
      <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
    </svg>
  );
}

function IconEdit({ size = 13 }: { size?: number }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7" />
      <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" />
    </svg>
  );
}

function IconSearch({ size = 14 }: { size?: number }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="11" cy="11" r="8" />
      <line x1="21" y1="21" x2="16.65" y2="16.65" />
    </svg>
  );
}

function IconGoogle({ size = 16 }: { size?: number }): JSX.Element {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24">
      <path
        fill="#EA4335"
        d="M12 5c1.6 0 3 .6 4.1 1.6l3.1-3.1C17.3 1.8 14.8 1 12 1 7.5 1 3.7 3.6 1.9 7.3l3.7 2.9C6.5 7.4 9 5 12 5z"
      />
      <path
        fill="#4285F4"
        d="M23.5 12.3c0-.8-.1-1.7-.2-2.3H12v4.6h6.5c-.3 1.5-1.1 2.8-2.4 3.7l3.7 2.9c2.2-2 3.7-5 3.7-8.9z"
      />
      <path
        fill="#FBBC05"
        d="M5.6 14.8c-.3-.8-.4-1.8-.4-2.8 0-1 .1-2 .4-2.8L1.9 6.3C.7 8.7 0 10.3 0 12s.7 3.3 1.9 5.7l3.7-2.9z"
      />
      <path
        fill="#34A853"
        d="M12 23c3.2 0 6-1.1 8-3l-3.7-2.9c-1.1.7-2.5 1.2-4.3 1.2-3 0-5.5-2.4-6.4-5.2L1.9 16c1.8 3.7 5.6 7 10.1 7z"
      />
    </svg>
  );
}

/**
 * Root component of the Job Cruiser Quick Fill side panel application.
 */
export function App(): JSX.Element {
  const [activeTab, setActiveTab] = useState<ActiveTab>('vault');
  const [profile, setProfile] = useState<QuickFillProfile>(DEFAULT_PROFILE);
  const [configuration, setConfiguration] = useState<ExtensionConfig>(DEFAULT_CONFIGURATION);
  const [notification, setNotification] = useState<NotificationState | null>(null);
  const [isSaving, setIsSaving] = useState<boolean>(false);
  const [isSyncing, setIsSyncing] = useState<boolean>(false);

  const displayNotification = useCallback((message: string, type: 'success' | 'error') => {
    setNotification({ message, type });
    setTimeout(() => {
      setNotification((current) => (current?.message === message ? null : current));
    }, 3500);
  }, []);

  const loadInitialData = useCallback(async () => {
    let currentConfiguration: ExtensionConfig = DEFAULT_CONFIGURATION;
    try {
      const fetchedConfig = (await browser.runtime.sendMessage({
        action: 'GET_CONFIG',
      })) as ExtensionConfig | undefined;
      if (fetchedConfig) {
        currentConfiguration = fetchedConfig;
        setConfiguration(fetchedConfig);
      }
    } catch {
      setConfiguration(DEFAULT_CONFIGURATION);
    }

    if (
      currentConfiguration.mode !== 'connected' ||
      !currentConfiguration.apiSecret?.trim()
    ) {
      try {
        const detectResult = (await browser.runtime.sendMessage({
          action: 'DETECT_WEB_SESSION',
        })) as { success?: boolean; token?: string } | undefined;
        if (detectResult?.success) {
          const autoConfig = (await browser.runtime.sendMessage({
            action: 'GET_CONFIG',
          })) as ExtensionConfig | undefined;
          if (autoConfig) {
            currentConfiguration = autoConfig;
            setConfiguration(autoConfig);
          }
        }
      } catch {
        return;
      }
    }

    try {
      const fetchedProfile = (await browser.runtime.sendMessage({
        action: 'GET_PROFILE',
      })) as QuickFillProfile | undefined;
      if (fetchedProfile) {
        setProfile(fetchedProfile);
      }
    } catch {
      setProfile(DEFAULT_PROFILE);
    }
  }, []);

  const handleOpenGoogleWebAuth = async () => {
    try {
      const detectResult = (await browser.runtime.sendMessage({
        action: 'DETECT_WEB_SESSION',
      })) as { success?: boolean; error?: string } | undefined;
      if (detectResult?.success) {
        await loadInitialData();
        setConfiguration((prev) => ({ ...prev, isSessionExpired: false }));
        displayNotification(
          'Active web session found! Connected and synced automatically.',
          'success',
        );
        return;
      }
    } catch {
      return;
    }

    const trimmedUrl = configuration.backendUrl.trim();
    const targetWebUrl = trimmedUrl.includes('web.app')
      ? trimmedUrl
      : DEFAULT_WEB_PORTAL_URL;

    void browser.tabs.create({ url: targetWebUrl });
    displayNotification(
      'Opening Job Cruiser Web Portal. Once signed in, Quick Fill connects automatically!',
      'success',
    );
  };

  const handleSyncFromBackend = useCallback(async () => {
    setIsSyncing(true);
    try {
      const syncResult = (await browser.runtime.sendMessage({
        action: 'SYNC_FROM_BACKEND',
      })) as { success: boolean; error?: string } | undefined;

      if (syncResult && syncResult.success) {
        await loadInitialData();
        setConfiguration((prev) => ({ ...prev, isSessionExpired: false }));
        displayNotification('Synchronized with Job Cruiser', 'success');
      } else {
        const failureReason = syncResult?.error ?? 'Synchronization failed';
        if (failureReason.includes('401')) {
          setConfiguration((prev) => ({ ...prev, isSessionExpired: true }));
        }
        displayNotification(failureReason, 'error');
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Network error during synchronization';
      displayNotification(errorMessage, 'error');
    } finally {
      setIsSyncing(false);
    }
  }, [loadInitialData, displayNotification]);

  useEffect(() => {
    void (async () => {
      await loadInitialData();
      const config = (await browser.runtime.sendMessage({ action: 'GET_CONFIG' })) as ExtensionConfig | undefined;
      if (config && config.mode === 'connected' && config.backendUrl && config.apiSecret) {
        void handleSyncFromBackend();
      }
    })();
  }, [loadInitialData, handleSyncFromBackend]);

  useEffect(() => {
    let lastHandledToken = '';
    const handleRuntimeMessage = (message: unknown) => {
      const extMsg = message as ExtensionMessage;
      if (extMsg.action === 'SESSION_CONNECTED') {
        const token = extMsg.payload?.token ?? '';
        if (token && token === lastHandledToken) {
          return;
        }
        lastHandledToken = token;
        setConfiguration((prev) => ({ ...prev, isSessionExpired: false }));
        void loadInitialData();
        displayNotification(
          'Connected and synced automatically from web session!',
          'success',
        );
      }
    };
    browser.runtime.onMessage.addListener(handleRuntimeMessage);
    return () => {
      browser.runtime.onMessage.removeListener(handleRuntimeMessage);
    };
  }, [loadInitialData, displayNotification]);

  const handleSaveProfile = async (explicitProfile?: QuickFillProfile) => {
    const profileToPersist = explicitProfile ?? profile;
    setIsSaving(true);
    try {
      const outcome = (await browser.runtime.sendMessage({
        action: 'SAVE_PROFILE',
        payload: profileToPersist,
      })) as { success?: boolean; error?: string } | undefined;
      if (outcome && outcome.error) {
        displayNotification(`Saved locally (${outcome.error})`, 'error');
      } else {
        displayNotification('Profile saved and synced', 'success');
      }
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to save profile';
      displayNotification(errorMessage, 'error');
    } finally {
      setIsSaving(false);
    }
  };

  const handleSaveConfiguration = async (updatedConfiguration: ExtensionConfig) => {
    setConfiguration(updatedConfiguration);
    try {
      await browser.runtime.sendMessage({
        action: 'SAVE_CONFIG',
        payload: updatedConfiguration,
      });
    } catch (error: unknown) {
      const errorMessage =
        error instanceof Error ? error.message : 'Failed to save configuration';
      displayNotification(errorMessage, 'error');
    }
  };

  const isExpired = Boolean(configuration.isSessionExpired);
  const isActuallyConnected =
    configuration.mode === 'connected' &&
    Boolean(configuration.apiSecret?.trim()) &&
    !isExpired;

  return (
    <div style={containerStyle}>
      <header style={headerStyle}>
        <div style={headerTitleRowStyle}>
          <div style={brandTitleStyle}>
            <IconBolt size={18} />
            <div>
              <div style={brandTextStyle}>Job Cruiser</div>
              <div style={brandSubtitleStyle}>Quick Fill Vault</div>
            </div>
          </div>

          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            {isActuallyConnected && (
              <button
                type="button"
                onClick={handleSyncFromBackend}
                disabled={isSyncing}
                title="Sync from Job Cruiser"
                style={iconRefreshButtonStyle}
              >
                <IconRotateCw size={13} isSpinning={isSyncing} />
              </button>
            )}

            <span
              style={{
                ...statusBadgeStyle,
                backgroundColor: isActuallyConnected
                  ? '#10B98115'
                  : isExpired
                    ? '#FEF2F2'
                    : '#F2F4F6',
                color: isActuallyConnected
                  ? '#059669'
                  : isExpired
                    ? '#DC2626'
                    : '#515F74',
                borderColor: isActuallyConnected
                  ? '#10B98140'
                  : isExpired
                    ? '#FCA5A5'
                    : '#C6C6CD',
                cursor: isExpired ? 'pointer' : 'default',
              }}
              onClick={isExpired ? handleOpenGoogleWebAuth : undefined}
              title={isExpired ? 'Session expired. Click to re-connect' : undefined}
            >
              {isActuallyConnected
                ? 'Connected'
                : isExpired
                  ? 'Expired — Re-connect'
                  : 'Local'}
            </span>
          </div>
        </div>

        <nav style={tabNavigationStyle}>
          <button
            type="button"
            onClick={() => setActiveTab('vault')}
            style={{
              ...tabButtonStyle,
              ...(activeTab === 'vault' ? activeTabButtonStyle : {}),
            }}
          >
            Vault
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('skills')}
            style={{
              ...tabButtonStyle,
              ...(activeTab === 'skills' ? activeTabButtonStyle : {}),
            }}
          >
            Skills
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('settings')}
            style={{
              ...tabButtonStyle,
              ...(activeTab === 'settings' ? activeTabButtonStyle : {}),
            }}
          >
            Settings
          </button>
        </nav>
      </header>

      {notification && (
        <div
          style={{
            ...notificationBannerStyle,
            backgroundColor:
              notification.type === 'success' ? '#ECFDF5' : '#FEF2F2',
            borderColor:
              notification.type === 'success' ? '#10B981' : '#BA1A1A',
            color: notification.type === 'success' ? '#065F46' : '#991B1B',
          }}
        >
          <span>{notification.message}</span>
          <button
            type="button"
            onClick={() => setNotification(null)}
            style={notificationCloseButtonStyle}
          >
            ✕
          </button>
        </div>
      )}

      <main style={mainContentStyle}>
        {activeTab === 'vault' && (
          <VaultEditor
            profile={profile}
            isSaving={isSaving}
            isConnected={isActuallyConnected}
            onProfileChange={setProfile}
            onSaveProfile={handleSaveProfile}
            onNotification={displayNotification}
            onOpenConnect={handleOpenGoogleWebAuth}
          />
        )}

        {activeTab === 'skills' && (
          <SkillsEditor
            profile={profile}
            configuration={configuration}
            onProfileChange={setProfile}
            onConfigurationChange={handleSaveConfiguration}
            onSaveProfile={handleSaveProfile}
            onNotification={displayNotification}
          />
        )}

        {activeTab === 'settings' && (
          <ConnectionSettings
            configuration={configuration}
            isSyncing={isSyncing}
            onConfigurationChange={handleSaveConfiguration}
            onSyncFromBackend={handleSyncFromBackend}
            onNotification={displayNotification}
            onReloadData={loadInitialData}
          />
        )}
      </main>
    </div>
  );
}

interface VaultEditorProperties {
  profile: QuickFillProfile;
  isSaving: boolean;
  isConnected: boolean;
  onProfileChange: (updatedProfile: QuickFillProfile) => void;
  onSaveProfile: (explicitProfile?: QuickFillProfile) => void;
  onNotification: (message: string, type: 'success' | 'error') => void;
  onOpenConnect: () => void;
}

const DEFAULT_CATEGORIES: readonly string[] = [
  'Personal & Contact',
  'Socials & Links',
  'Work Authorization',
  'Experience & Education',
  'Custom Answers',
];

/**
 * Copies specified text to system clipboard and dispatches an insertion command to the active browser tab
 * to inject text directly at the current cursor position.
 *
 * @param text - The text payload to copy and inject.
 */
async function copyAndInjectAtCursor(text: string): Promise<void> {
  try {
    await navigator.clipboard.writeText(text);
  } catch {}

  try {
    const tabs = await browser.tabs.query({ active: true, currentWindow: true });
    const targetTab = tabs[0];
    if (targetTab?.id) {
      await browser.tabs.sendMessage(targetTab.id, {
        action: 'INJECT_AT_CURSOR',
        payload: { text },
      });
    }
  } catch {}
}

/**
 * Formats all candidate profile fields into a unified clipboard-ready summary text block.
 */
function buildEverythingSummary(profile: QuickFillProfile): string {
  const deleted = new Set(profile.deletedFieldIds ?? []);
  const lines: string[] = [];
  if (profile.fullName) lines.push(`Full Name: ${profile.fullName}`);
  if (profile.email && !deleted.has('email') && !deleted.has('profile_email')) lines.push(`Email Address: ${profile.email}`);
  if (profile.phone && !deleted.has('phone') && !deleted.has('profile_phone')) lines.push(`Phone Number: ${profile.phone}`);
  if (profile.location && !deleted.has('location') && !deleted.has('profile_location')) lines.push(`Current Location: ${profile.location}`);
  if (profile.country && !deleted.has('country') && !deleted.has('profile_country')) lines.push(`Country: ${profile.country}`);
  if (profile.professionalHeadline && !deleted.has('professionalHeadline') && !deleted.has('profile_headline')) lines.push(`Professional Headline: ${profile.professionalHeadline}`);
  if (profile.linkedinUrl && !deleted.has('linkedinUrl') && !deleted.has('link_linkedin')) lines.push(`LinkedIn URL: ${profile.linkedinUrl}`);
  if (profile.githubUrl && !deleted.has('githubUrl') && !deleted.has('link_github')) lines.push(`GitHub URL: ${profile.githubUrl}`);
  if (profile.portfolioUrl && !deleted.has('portfolioUrl') && !deleted.has('link_portfolio')) lines.push(`Portfolio URL: ${profile.portfolioUrl}`);
  for (const link of profile.customLinks ?? []) {
    if (link.url) lines.push(`${link.label || 'Link'}: ${link.url}`);
  }
  if (profile.workAuthorization && !deleted.has('workAuthorization') && !deleted.has('qa_work_authorization')) lines.push(`Work Authorization: ${profile.workAuthorization}`);
  if (profile.visaSponsorship && !deleted.has('visaSponsorship') && !deleted.has('qa_visa_sponsorship')) lines.push(`Visa Sponsorship: ${profile.visaSponsorship}`);
  if (profile.noticePeriod && !deleted.has('noticePeriod') && !deleted.has('qa_notice_period')) lines.push(`Notice Period: ${profile.noticePeriod}`);
  if (profile.willingToRelocate && !deleted.has('willingToRelocate') && !deleted.has('qa_willing_to_relocate')) lines.push(`Willing to Relocate: ${profile.willingToRelocate}`);
  if (profile.expectedSalary && !deleted.has('expectedSalary') && !deleted.has('qa_expected_salary')) lines.push(`Expected Salary: ${profile.expectedSalary}`);
  if (profile.currentSalary && !deleted.has('currentSalary') && !deleted.has('qa_current_salary')) lines.push(`Current Salary: ${profile.currentSalary}`);
  if (profile.currentRole && !deleted.has('currentRole') && !deleted.has('exp_current_role')) lines.push(`Current Role: ${profile.currentRole}`);
  if (profile.currentCompany && !deleted.has('currentCompany') && !deleted.has('exp_current_company')) lines.push(`Current Company: ${profile.currentCompany}`);
  if (profile.highestDegree && !deleted.has('highestDegree') && !deleted.has('edu_degree')) lines.push(`Highest Degree: ${profile.highestDegree}`);
  if (profile.institution && !deleted.has('institution') && !deleted.has('edu_institution')) lines.push(`University / Institution: ${profile.institution}`);
  if (profile.graduationYear && !deleted.has('graduationYear') && !deleted.has('edu_year')) lines.push(`Graduation Year: ${profile.graduationYear}`);
  if (profile.skills && profile.skills.length > 0) lines.push(`Technical Skills: ${profile.skills.join(', ')}`);
  if (profile.bioSummary && !deleted.has('bioSummary') && !deleted.has('bio_summary')) lines.push(`Professional Bio: ${profile.bioSummary}`);
  for (const field of profile.customFields ?? []) {
    if (field.value && !deleted.has(field.id)) lines.push(`${field.label || 'Custom Answer'}: ${field.value}`);
  }
  return lines.join('\n');
}

interface QuickFillCardProperties {
  id: string;
  label: string;
  value: string;
  category: string;
  isCopied: boolean;
  isMultiline?: boolean;
  isCustom?: boolean;
  onCopy: () => void;
  onSave: (newLabel: string, newValue: string) => void;
  onDelete?: () => void;
}

/**
 * 1-click copy card component replicating Job Cruiser's quick fill card interaction.
 * Clicking anywhere on the card copies the value; clicking the corner edit icon toggles inline editing.
 */
function QuickFillCard({
  label,
  value,
  category,
  isCopied,
  isMultiline = false,
  isCustom = false,
  onCopy,
  onSave,
  onDelete,
}: QuickFillCardProperties): JSX.Element {
  const [isEditing, setIsEditing] = useState<boolean>(false);
  const [isExpanded, setIsExpanded] = useState<boolean>(false);
  const [editLabel, setEditLabel] = useState<string>(label);
  const [editValue, setEditValue] = useState<string>(value);

  const hasMoreContent = value.length > 40 || value.includes('\n');

  const handleStartEdit = (event: JSX.TargetedMouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    setEditLabel(label);
    setEditValue(value);
    setIsEditing(true);
  };

  const handleCancelEdit = (event: JSX.TargetedMouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    setIsEditing(false);
    setEditLabel(label);
    setEditValue(value);
  };

  const handleCommitSave = (event: JSX.TargetedMouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    onSave(editLabel.trim() || label, editValue.trim());
    setIsEditing(false);
  };

  const handleDeleteClick = (event: JSX.TargetedMouseEvent<HTMLButtonElement>) => {
    event.stopPropagation();
    if (onDelete) {
      onDelete();
    }
  };

  if (isEditing) {
    return (
      <div
        style={quickFillCardEditingContainerStyle}
        onClick={(event) => event.stopPropagation()}
      >
        <div style={quickFillCardHeaderRowStyle}>
          {isCustom ? (
            <input
              type="text"
              value={editLabel}
              onInput={(event) =>
                setEditLabel((event.target as HTMLInputElement).value)
              }
              placeholder="Field Label"
              style={compactInputStyle}
            />
          ) : (
            <span style={quickFillCardLabelTextStyle}>Edit {label}</span>
          )}
          <span style={quickFillCardCategoryBadgeStyle}>{category}</span>
        </div>

        <div style={{ marginTop: '8px' }}>
          {isMultiline ? (
            <textarea
              value={editValue}
              onInput={(event) =>
                setEditValue((event.target as HTMLTextAreaElement).value)
              }
              placeholder={`Enter ${label}...`}
              style={textareaInputStyle}
            />
          ) : (
            <input
              type="text"
              value={editValue}
              onInput={(event) =>
                setEditValue((event.target as HTMLInputElement).value)
              }
              placeholder={`Enter ${label}...`}
              style={textInputStyle}
            />
          )}
        </div>

        <div style={quickFillCardEditActionsRowStyle}>
          <button
            type="button"
            onClick={handleCancelEdit}
            style={secondaryButtonStyle}
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleCommitSave}
            style={compactPrimaryButtonStyle}
          >
            Save
          </button>
        </div>
      </div>
    );
  }

  return (
    <div
      style={{
        ...quickFillCardContainerStyle,
        borderColor: isCopied ? '#10B981' : '#E2E8F0',
        borderWidth: isCopied ? '1.5px' : '1px',
      }}
      onClick={onCopy}
      title={`Click anywhere to copy ${label}`}
    >
      <div style={quickFillCardHeaderRowStyle}>
        <div style={quickFillCardLabelBoxStyle}>
          <span style={quickFillCardLabelTextStyle}>{label}</span>
          <span style={quickFillCardCategoryBadgeStyle}>{category}</span>
        </div>

        <div style={quickFillCardActionButtonsBoxStyle}>
          <button
            type="button"
            onClick={handleStartEdit}
            title={`Edit ${label}`}
            style={quickFillIconActionButtonStyle}
          >
            <IconEdit size={13} />
          </button>

          {onDelete && (
            <button
              type="button"
              onClick={handleDeleteClick}
              title={`Delete ${label}`}
              style={deleteIconButtonStyle}
            >
              <IconTrash size={13} />
            </button>
          )}

          <div
            style={{
              ...quickFillCopyBadgeStyle,
              backgroundColor: isCopied ? '#10B981' : '#F2F4F6',
              color: isCopied ? '#FFFFFF' : '#515F74',
            }}
          >
            {isCopied ? (
              <IconCheck size={12} color="#FFFFFF" />
            ) : (
              <IconCopy size={12} />
            )}
            <span>{isCopied ? 'Copied' : 'Copy'}</span>
          </div>
        </div>
      </div>

      <div style={quickFillCardValueBoxStyle}>
        <div style={{ flex: 1, minWidth: 0, overflow: 'hidden' }}>
          <span
            style={{
              ...quickFillCardValueTextStyle,
              display: 'block',
              overflow: isExpanded ? 'visible' : 'hidden',
              textOverflow: isExpanded ? 'clip' : 'ellipsis',
              whiteSpace: isExpanded ? 'pre-wrap' : 'nowrap',
              wordBreak: isExpanded ? 'break-word' : 'normal',
            }}
          >
            {value ? (
              isExpanded ? value : value.replaceAll('\n', ' ').trim()
            ) : (
              <span style={emptyStateTextStyle}>
                (Empty — click pencil in corner to set)
              </span>
            )}
          </span>
        </div>
        {hasMoreContent && (
          <button
            type="button"
            onClick={(event) => {
              event.stopPropagation();
              setIsExpanded(!isExpanded);
            }}
            style={expandToggleButtonStyle}
            title={isExpanded ? 'Collapse content' : 'Expand full content'}
          >
            <span>{isExpanded ? 'Less' : 'More'}</span>
            <span style={{ fontSize: '10px' }}>{isExpanded ? '▴' : '▾'}</span>
          </button>
        )}
      </div>
    </div>
  );
}

/**
 * Vault editor providing 1-click clipboard copy cards, corner editing, global copying, and categorization.
 */
function VaultEditor({
  profile,
  isConnected,
  onProfileChange,
  onSaveProfile,
  onNotification,
  onOpenConnect,
}: VaultEditorProperties): JSX.Element {
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [showAddCustom, setShowAddCustom] = useState<boolean>(false);
  const [newCustomLabel, setNewCustomLabel] = useState<string>('');
  const [newCustomValue, setNewCustomValue] = useState<string>('');
  const [newCustomCategory, setNewCustomCategory] = useState<string>('Custom Answers');
  const [isCustomCategoryMode, setIsCustomCategoryMode] = useState<boolean>(false);
  const [customCategoryInput, setCustomCategoryInput] = useState<string>('');

  const availableCategories = Array.from(
    new Set([
      ...DEFAULT_CATEGORIES,
      ...(profile.customFields ?? [])
        .map((field) => field.category?.trim())
        .filter((cat): cat is string => Boolean(cat && cat.length > 0)),
    ]),
  );

  const categoryFilterOptions = ['All', ...availableCategories];

  const copyToClipboard = (key: string, value: string | undefined, label: string) => {
    if (!value || !value.trim()) {
      onNotification(`No value set for ${label}. Click pencil to edit.`, 'error');
      return;
    }
    void copyAndInjectAtCursor(value.trim());
    setCopiedKey(key);
    setTimeout(() => {
      setCopiedKey((curr) => (curr === key ? null : curr));
    }, 1600);
    onNotification(`Copied & inserted "${label}"`, 'success');
  };

  const handleCopyEverything = () => {
    const summary = buildEverythingSummary(profile);
    if (!summary.trim()) {
      onNotification('Profile has no fields to copy', 'error');
      return;
    }
    void copyAndInjectAtCursor(summary);
    setCopiedKey('everything');
    setTimeout(() => {
      setCopiedKey((curr) => (curr === 'everything' ? null : curr));
    }, 1800);
    onNotification('Copied all candidate fields to clipboard!', 'success');
  };

  const handleSaveStandardField = (fieldKey: keyof QuickFillProfile, newValue: string) => {
    let updatedProfile: QuickFillProfile;
    if (fieldKey === 'fullName') {
      const parts = newValue.trim().split(/\s+/);
      const firstName = parts[0] ?? '';
      const lastName = parts.slice(1).join(' ');
      updatedProfile = {
        ...profile,
        fullName: newValue,
        firstName,
        lastName,
      };
    } else {
      updatedProfile = {
        ...profile,
        [fieldKey]: newValue,
      };
    }
    onProfileChange(updatedProfile);
    onSaveProfile(updatedProfile);
  };

  const handleDeleteStandardField = (
    fieldKey: keyof QuickFillProfile,
    jobCruiserId: string,
    label: string,
  ) => {
    const currentDeleted = new Set(profile.deletedFieldIds ?? []);
    currentDeleted.add(fieldKey as string);
    currentDeleted.add(jobCruiserId);
    const updatedProfile: QuickFillProfile = {
      ...profile,
      [fieldKey]: '',
      deletedFieldIds: Array.from(currentDeleted),
    };
    onProfileChange(updatedProfile);
    onSaveProfile(updatedProfile);
    onNotification(`Deleted "${label}"`, 'success');
  };

  const handleSaveCustomLink = (index: number, newLabel: string, newUrl: string) => {
    const updated = [...(profile.customLinks ?? [])];
    updated[index] = { label: newLabel, url: newUrl };
    const updatedProfile = {
      ...profile,
      customLinks: updated,
    };
    onProfileChange(updatedProfile);
    onSaveProfile(updatedProfile);
  };

  const handleDeleteCustomLink = (index: number) => {
    const updated = (profile.customLinks ?? []).filter((_, i) => i !== index);
    const updatedProfile = {
      ...profile,
      customLinks: updated,
    };
    onProfileChange(updatedProfile);
    onSaveProfile(updatedProfile);
  };

  const handleSaveCustomField = (index: number, newLabel: string, newValue: string) => {
    const updated = [...(profile.customFields ?? [])];
    const existing = updated[index];
    if (existing) {
      updated[index] = { ...existing, label: newLabel, value: newValue };
      const updatedProfile = {
        ...profile,
        customFields: updated,
      };
      onProfileChange(updatedProfile);
      onSaveProfile(updatedProfile);
    }
  };

  const handleDeleteCustomField = (index: number) => {
    const target = (profile.customFields ?? [])[index];
    const updated = (profile.customFields ?? []).filter((_, i) => i !== index);
    const currentDeleted = new Set(profile.deletedFieldIds ?? []);
    if (target?.id) {
      currentDeleted.add(target.id);
    }
    const updatedProfile = {
      ...profile,
      customFields: updated,
      deletedFieldIds: Array.from(currentDeleted),
    };
    onProfileChange(updatedProfile);
    onSaveProfile(updatedProfile);
  };

  const handleAddCustom = () => {
    const trimmedLabel = newCustomLabel.trim();
    const trimmedValue = newCustomValue.trim();
    if (!trimmedLabel || !trimmedValue) {
      onNotification('Please enter both label and value', 'error');
      return;
    }

    const currentDeleted = (profile.deletedFieldIds ?? []).filter(
      (id) => id.toLowerCase() !== trimmedLabel.toLowerCase(),
    );

    const targetCategory = isCustomCategoryMode
      ? customCategoryInput.trim() || 'Custom Answers'
      : newCustomCategory;

    let updatedProfile: QuickFillProfile;
    if (targetCategory === 'Socials & Links') {
      const currentLinks = profile.customLinks ?? [];
      updatedProfile = {
        ...profile,
        customLinks: [...currentLinks, { label: trimmedLabel, url: trimmedValue }],
        deletedFieldIds: currentDeleted,
      };
    } else {
      const currentFields = profile.customFields ?? [];
      const newField: CustomFormField = {
        id: `custom_${Date.now()}`,
        label: trimmedLabel,
        value: trimmedValue,
        category: targetCategory,
      };
      updatedProfile = {
        ...profile,
        customFields: [...currentFields, newField],
        deletedFieldIds: currentDeleted,
      };
    }

    onProfileChange(updatedProfile);
    setNewCustomLabel('');
    setNewCustomValue('');
    setCustomCategoryInput('');
    setIsCustomCategoryMode(false);
    setShowAddCustom(false);
    onSaveProfile(updatedProfile);
    onNotification(`Added "${trimmedLabel}" in "${targetCategory}"`, 'success');
  };

  const isFieldDeleted = (fieldId: string, jobCruiserId?: string): boolean => {
    const deletedList = profile.deletedFieldIds ?? [];
    if (deletedList.includes(fieldId)) {
      return true;
    }
    if (jobCruiserId && deletedList.includes(jobCruiserId)) {
      return true;
    }
    return false;
  };

  interface ItemEntry {
    id: string;
    jobCruiserId?: string;
    label: string;
    value: string;
    category: string;
    isMultiline?: boolean;
    isCustom?: boolean;
    onSave: (newLabel: string, newValue: string) => void;
    onDelete?: () => void;
  }

  const allItems: ItemEntry[] = [
    {
      id: 'fullName',
      jobCruiserId: 'profile_full_name',
      label: 'Full Name',
      value: profile.fullName ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('fullName', val),
      onDelete: () => handleDeleteStandardField('fullName', 'profile_full_name', 'Full Name'),
    },
    {
      id: 'email',
      jobCruiserId: 'profile_email',
      label: 'Email Address',
      value: profile.email ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('email', val),
      onDelete: () => handleDeleteStandardField('email', 'profile_email', 'Email Address'),
    },
    {
      id: 'phone',
      jobCruiserId: 'profile_phone',
      label: 'Phone Number',
      value: profile.phone ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('phone', val),
      onDelete: () => handleDeleteStandardField('phone', 'profile_phone', 'Phone Number'),
    },
    {
      id: 'location',
      jobCruiserId: 'profile_location',
      label: 'Current Location',
      value: profile.location ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('location', val),
      onDelete: () => handleDeleteStandardField('location', 'profile_location', 'Current Location'),
    },
    {
      id: 'country',
      jobCruiserId: 'profile_country',
      label: 'Country',
      value: profile.country ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('country', val),
      onDelete: () => handleDeleteStandardField('country', 'profile_country', 'Country'),
    },
    {
      id: 'professionalHeadline',
      jobCruiserId: 'profile_headline',
      label: 'Professional Headline',
      value: profile.professionalHeadline ?? '',
      category: 'Personal & Contact',
      onSave: (_, val) => handleSaveStandardField('professionalHeadline', val),
      onDelete: () => handleDeleteStandardField('professionalHeadline', 'profile_headline', 'Professional Headline'),
    },
    {
      id: 'linkedinUrl',
      jobCruiserId: 'link_linkedin',
      label: 'LinkedIn Profile URL',
      value: profile.linkedinUrl ?? '',
      category: 'Socials & Links',
      onSave: (_, val) => handleSaveStandardField('linkedinUrl', val),
      onDelete: () => handleDeleteStandardField('linkedinUrl', 'link_linkedin', 'LinkedIn Profile URL'),
    },
    {
      id: 'githubUrl',
      jobCruiserId: 'link_github',
      label: 'GitHub Profile URL',
      value: profile.githubUrl ?? '',
      category: 'Socials & Links',
      onSave: (_, val) => handleSaveStandardField('githubUrl', val),
      onDelete: () => handleDeleteStandardField('githubUrl', 'link_github', 'GitHub Profile URL'),
    },
    {
      id: 'portfolioUrl',
      jobCruiserId: 'link_portfolio',
      label: 'Portfolio / Website URL',
      value: profile.portfolioUrl ?? '',
      category: 'Socials & Links',
      onSave: (_, val) => handleSaveStandardField('portfolioUrl', val),
      onDelete: () => handleDeleteStandardField('portfolioUrl', 'link_portfolio', 'Portfolio / Website URL'),
    },
    ...(profile.customLinks ?? []).map((link, idx) => ({
      id: `link_${idx}`,
      label: link.label || 'Custom Link',
      value: link.url,
      category: 'Socials & Links',
      isCustom: true,
      onSave: (lbl: string, val: string) => handleSaveCustomLink(idx, lbl, val),
      onDelete: () => handleDeleteCustomLink(idx),
    })),
    {
      id: 'workAuthorization',
      jobCruiserId: 'qa_work_authorization',
      label: 'Work Authorization',
      value: profile.workAuthorization ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('workAuthorization', val),
      onDelete: () => handleDeleteStandardField('workAuthorization', 'qa_work_authorization', 'Work Authorization'),
    },
    {
      id: 'visaSponsorship',
      jobCruiserId: 'qa_visa_sponsorship',
      label: 'Visa Sponsorship Needed?',
      value: profile.visaSponsorship ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('visaSponsorship', val),
      onDelete: () => handleDeleteStandardField('visaSponsorship', 'qa_visa_sponsorship', 'Visa Sponsorship Needed?'),
    },
    {
      id: 'noticePeriod',
      jobCruiserId: 'qa_notice_period',
      label: 'Notice Period',
      value: profile.noticePeriod ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('noticePeriod', val),
      onDelete: () => handleDeleteStandardField('noticePeriod', 'qa_notice_period', 'Notice Period'),
    },
    {
      id: 'willingToRelocate',
      jobCruiserId: 'qa_willing_to_relocate',
      label: 'Willing to Relocate?',
      value: profile.willingToRelocate ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('willingToRelocate', val),
      onDelete: () => handleDeleteStandardField('willingToRelocate', 'qa_willing_to_relocate', 'Willing to Relocate?'),
    },
    {
      id: 'expectedSalary',
      jobCruiserId: 'qa_expected_salary',
      label: 'Expected Salary',
      value: profile.expectedSalary ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('expectedSalary', val),
      onDelete: () => handleDeleteStandardField('expectedSalary', 'qa_expected_salary', 'Expected Salary'),
    },
    {
      id: 'currentSalary',
      jobCruiserId: 'qa_current_salary',
      label: 'Current Salary',
      value: profile.currentSalary ?? '',
      category: 'Work Authorization',
      onSave: (_, val) => handleSaveStandardField('currentSalary', val),
      onDelete: () => handleDeleteStandardField('currentSalary', 'qa_current_salary', 'Current Salary'),
    },
    {
      id: 'currentRole',
      jobCruiserId: 'exp_current_role',
      label: 'Current / Most Recent Role',
      value: profile.currentRole ?? '',
      category: 'Experience & Education',
      onSave: (_, val) => handleSaveStandardField('currentRole', val),
      onDelete: () => handleDeleteStandardField('currentRole', 'exp_current_role', 'Current / Most Recent Role'),
    },
    {
      id: 'currentCompany',
      jobCruiserId: 'exp_current_company',
      label: 'Current / Most Recent Company',
      value: profile.currentCompany ?? '',
      category: 'Experience & Education',
      onSave: (_, val) => handleSaveStandardField('currentCompany', val),
      onDelete: () => handleDeleteStandardField('currentCompany', 'exp_current_company', 'Current / Most Recent Company'),
    },
    {
      id: 'highestDegree',
      jobCruiserId: 'edu_degree',
      label: 'Highest Degree',
      value: profile.highestDegree ?? '',
      category: 'Experience & Education',
      onSave: (_, val) => handleSaveStandardField('highestDegree', val),
      onDelete: () => handleDeleteStandardField('highestDegree', 'edu_degree', 'Highest Degree'),
    },
    {
      id: 'institution',
      jobCruiserId: 'edu_institution',
      label: 'University / Institution',
      value: profile.institution ?? '',
      category: 'Experience & Education',
      onSave: (_, val) => handleSaveStandardField('institution', val),
      onDelete: () => handleDeleteStandardField('institution', 'edu_institution', 'University / Institution'),
    },
    {
      id: 'graduationYear',
      jobCruiserId: 'edu_year',
      label: 'Graduation Year',
      value: profile.graduationYear ?? '',
      category: 'Experience & Education',
      onSave: (_, val) => handleSaveStandardField('graduationYear', val),
      onDelete: () => handleDeleteStandardField('graduationYear', 'edu_year', 'Graduation Year'),
    },
    {
      id: 'bioSummary',
      jobCruiserId: 'bio_summary',
      label: 'Professional Bio',
      value: profile.bioSummary ?? '',
      category: 'Custom Answers',
      isMultiline: true,
      onSave: (_, val) => handleSaveStandardField('bioSummary', val),
      onDelete: () => handleDeleteStandardField('bioSummary', 'bio_summary', 'Professional Bio'),
    },
    ...(profile.skills && profile.skills.length > 0
      ? [
          {
            id: 'skills',
            jobCruiserId: 'profile_skills',
            label: 'Technical Skills',
            value: profile.skills.join(', '),
            category: 'Experience & Education',
            isMultiline: true,
            onSave: (_: string, val: string) => {
              const updatedSkills = val
                .split(',')
                .map((s) => s.trim())
                .filter(Boolean);
              const updatedProfile = {
                ...profile,
                skills: updatedSkills,
              };
              onProfileChange(updatedProfile);
              onSaveProfile(updatedProfile);
            },
            onDelete: () => {
              const updatedProfile = {
                ...profile,
                skills: [],
              };
              onProfileChange(updatedProfile);
              onSaveProfile(updatedProfile);
            },
          },
        ]
      : []),
    ...(profile.customFields ?? []).map((field, idx) => ({
      id: field.id || `custom_${idx}`,
      label: field.label || 'Custom Answer',
      value: field.value,
      category: field.category || 'Custom Answers',
      isCustom: true,
      isMultiline: field.value.length > 40 || field.value.includes('\n'),
      onSave: (lbl: string, val: string) => handleSaveCustomField(idx, lbl, val),
      onDelete: () => handleDeleteCustomField(idx),
    })),
  ];

  const query = searchQuery.trim().toLowerCase();
  const visibleItems = allItems.filter(
    (item) =>
      !isFieldDeleted(item.id, item.jobCruiserId) &&
      Boolean(item.value && item.value.trim()),
  );
  const filteredItems = visibleItems.filter((item) => {
    const matchesCategory =
      selectedCategory === 'All' || item.category === selectedCategory;
    if (!matchesCategory) return false;
    if (!query) return true;
    return (
      item.label.toLowerCase().includes(query) ||
      item.value.toLowerCase().includes(query)
    );
  });

  return (
    <div style={vaultContainerStyle}>
      {!isConnected && (
        <div style={disconnectedBannerCardStyle}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <span style={{ fontWeight: 700, fontSize: '12px', color: '#191C1E' }}>
              Connect Job Cruiser Web Portal
            </span>
            <span style={quickFillCardCategoryBadgeStyle}>Local Mode</span>
          </div>
          <p style={helpTextStyle}>
            Sync your profile and tailored skills with 1 tap via Google authentication.
          </p>
          <button
            type="button"
            onClick={onOpenConnect}
            style={{ ...googleButtonStyle, marginTop: '4px', padding: '7px 12px' }}
          >
            <IconGoogle size={14} />
            <span>Sign in with Google (Web Portal)</span>
          </button>
        </div>
      )}

      <div style={topActionBarRowStyle}>
        <button
          type="button"
          onClick={handleCopyEverything}
          style={{
            ...copyEverythingButtonStyle,
            backgroundColor: copiedKey === 'everything' ? '#10B981' : '#000000',
          }}
          title="Copy all candidate profile fields to clipboard"
        >
          {copiedKey === 'everything' ? (
            <>
              <IconCheck size={14} color="#FFFFFF" />
              <span>Copied Everything!</span>
            </>
          ) : (
            <>
              <IconCopy size={14} />
              <span>Copy Everything</span>
            </>
          )}
        </button>

        <button
          type="button"
          onClick={() => setShowAddCustom(!showAddCustom)}
          style={secondaryButtonStyle}
          title="Add a custom answer or link"
        >
          {showAddCustom ? 'Close' : '+ Add Field'}
        </button>
      </div>

      {showAddCustom && (
        <div style={quickFillCardEditingContainerStyle}>
          <span style={quickFillCardLabelTextStyle}>Add Custom Field</span>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '6px' }}>
            <input
              type="text"
              placeholder="Field Label / Question (e.g. Desired Start Date)"
              value={newCustomLabel}
              onInput={(e) =>
                setNewCustomLabel((e.target as HTMLInputElement).value)
              }
              style={textInputStyle}
            />
            <textarea
              placeholder="Your Answer / Value / URL"
              value={newCustomValue}
              onInput={(e) =>
                setNewCustomValue((e.target as HTMLTextAreaElement).value)
              }
              style={{ ...textareaInputStyle, minHeight: '60px' }}
            />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <label style={fieldLabelStyle}>Category</label>
              <button
                type="button"
                onClick={() => setIsCustomCategoryMode(!isCustomCategoryMode)}
                style={linkToggleButtonStyle}
              >
                {isCustomCategoryMode ? 'Choose Existing' : '+ New Category'}
              </button>
            </div>
            {isCustomCategoryMode ? (
              <input
                type="text"
                placeholder="Custom Category Name (e.g. Pitches, Clearances)"
                value={customCategoryInput}
                onInput={(e) =>
                  setCustomCategoryInput((e.target as HTMLInputElement).value)
                }
                style={textInputStyle}
              />
            ) : (
              <select
                value={newCustomCategory}
                onChange={(e) =>
                  setNewCustomCategory((e.target as HTMLSelectElement).value)
                }
                style={{ ...compactInputStyle, width: '100%' }}
              >
                {availableCategories.map((categoryName) => (
                  <option key={categoryName} value={categoryName}>
                    {categoryName}
                  </option>
                ))}
              </select>
            )}
            <div style={quickFillCardEditActionsRowStyle}>
              <button
                type="button"
                onClick={() => {
                  setShowAddCustom(false);
                  setIsCustomCategoryMode(false);
                  setCustomCategoryInput('');
                }}
                style={secondaryButtonStyle}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleAddCustom}
                style={compactPrimaryButtonStyle}
              >
                Add Field
              </button>
            </div>
          </div>
        </div>
      )}

      <div style={quickFillSearchContainerStyle}>
        <div style={quickFillSearchIconStyle}>
          <IconSearch size={14} />
        </div>
        <input
          type="text"
          placeholder="Search fields or values (e.g. visa, salary, email)..."
          value={searchQuery}
          onInput={(e) =>
            setSearchQuery((e.target as HTMLInputElement).value)
          }
          style={quickFillSearchInputStyle}
        />
        {searchQuery && (
          <button
            type="button"
            onClick={() => setSearchQuery('')}
            style={quickFillClearSearchButtonStyle}
            title="Clear search"
          >
            ✕
          </button>
        )}
      </div>

      <div style={quickFillCategoryChipsContainerStyle}>
        {categoryFilterOptions.map((cat) => {
          const isSelected = selectedCategory === cat;
          return (
            <button
              key={cat}
              type="button"
              onClick={() => setSelectedCategory(cat)}
              style={{
                ...quickFillCategoryChipStyle,
                backgroundColor: isSelected ? '#000000' : '#FFFFFF',
                color: isSelected ? '#FFFFFF' : '#515F74',
                borderColor: isSelected ? '#000000' : '#E2E8F0',
              }}
            >
              {cat}
            </button>
          );
        })}
      </div>

      {filteredItems.length === 0 ? (
        <div style={{ ...quickFillCardContainerStyle, alignItems: 'center', textAlign: 'center', padding: '24px' }}>
          <p style={emptyStateTextStyle}>
            {visibleItems.length === 0
              ? 'No candidate fields saved yet. Connect to Job Cruiser to sync or click "+ Add Field" above.'
              : `No fields found matching "${searchQuery || selectedCategory}".`}
          </p>
        </div>
      ) : (
        filteredItems.map((item) => (
          <QuickFillCard
            key={item.id}
            id={item.id}
            label={item.label}
            value={item.value}
            category={item.category}
            isCopied={copiedKey === item.id}
            isMultiline={item.isMultiline}
            isCustom={item.isCustom}
            onCopy={() => copyToClipboard(item.id, item.value, item.label)}
            onSave={item.onSave}
            onDelete={item.onDelete}
          />
        ))
      )}
    </div>
  );
}

interface SkillsEditorProperties {
  profile: QuickFillProfile;
  configuration: ExtensionConfig;
  onProfileChange: (updatedProfile: QuickFillProfile) => void;
  onConfigurationChange: (updatedConfig: ExtensionConfig) => void;
  onSaveProfile: (explicitProfile?: QuickFillProfile) => void;
  onNotification?: (message: string, type: 'success' | 'error') => void;
}

/**
 * Skills editor component matching Job Cruiser design.
 */
function SkillsEditor({
  profile,
  configuration,
  onProfileChange,
  onConfigurationChange,
  onSaveProfile,
  onNotification,
}: SkillsEditorProperties): JSX.Element {
  const [newSkillText, setNewSkillText] = useState<string>('');
  const [activeSkillIndex, setActiveSkillIndex] = useState<number>(0);
  const [copiedSkill, setCopiedSkill] = useState<string | null>(null);

  const skills = profile.skills ?? [];
  const skillsCount = skills.length;
  const safeIndex = skillsCount > 0 ? Math.min(activeSkillIndex, skillsCount - 1) : 0;
  const currentSkill = skills[safeIndex] ?? '';
  const nextSkillIndex = (safeIndex + 1) % (skillsCount || 1);
  const nextSkill = skills[nextSkillIndex] ?? '';

  const handleCopyCurrentAndAdvance = () => {
    if (!currentSkill) return;
    void copyAndInjectAtCursor(currentSkill);
    setCopiedSkill(currentSkill);
    setTimeout(() => {
      setCopiedSkill((curr) => (curr === currentSkill ? null : curr));
    }, 1500);

    const advancedIndex = (safeIndex + 1) % skillsCount;
    setActiveSkillIndex(advancedIndex);

    if (onNotification) {
      if (skillsCount > 1) {
        onNotification(`Copied "${currentSkill}"! Next is "${nextSkill}"`, 'success');
      } else {
        onNotification(`Copied "${currentSkill}"`, 'success');
      }
    }
  };

  const handleCopySpecificSkill = (skill: string, index: number) => {
    void copyAndInjectAtCursor(skill);
    setCopiedSkill(skill);
    setActiveSkillIndex((index + 1) % skillsCount);
    setTimeout(() => {
      setCopiedSkill((curr) => (curr === skill ? null : curr));
    }, 1500);
    if (onNotification) {
      onNotification(`Copied "${skill}"`, 'success');
    }
  };

  const handleStepPrev = () => {
    if (skillsCount === 0) return;
    setActiveSkillIndex((safeIndex - 1 + skillsCount) % skillsCount);
  };

  const handleStepNext = () => {
    if (skillsCount === 0) return;
    setActiveSkillIndex((safeIndex + 1) % skillsCount);
  };

  const handleResetStepper = () => {
    setActiveSkillIndex(0);
  };

  const handleAddSkill = () => {
    const trimmed = newSkillText.trim();
    if (!trimmed) return;

    const currentSkills = profile.skills ?? [];
    if (!currentSkills.includes(trimmed)) {
      const updated = {
        ...profile,
        skills: [...currentSkills, trimmed],
      };
      onProfileChange(updated);
      onSaveProfile(updated);
    }
    setNewSkillText('');
  };

  const handleRemoveSkill = (skillToRemove: string) => {
    const currentSkills = (profile.skills ?? []).filter(
      (skill) => skill !== skillToRemove,
    );
    const updated = {
      ...profile,
      skills: currentSkills,
    };
    onProfileChange(updated);
    onSaveProfile(updated);
  };

  const handleKeyDown = (event: JSX.TargetedKeyboardEvent<HTMLInputElement>) => {
    if (event.key === 'Enter') {
      event.preventDefault();
      handleAddSkill();
    }
  };

  const handleModeChange = (mode: 'cycle_one_by_one' | 'dump_all') => {
    onConfigurationChange({
      ...configuration,
      skillsDefaultMode: mode,
    });
  };

  return (
    <div style={skillsContainerStyle}>
      <div style={topActionBarStyle}>
        <button
          type="button"
          onClick={() => onSaveProfile(profile)}
          style={primaryButtonStyle}
        >
          Save Skills
        </button>
      </div>

      {skillsCount > 0 && (
        <div style={cardContainerStyle}>
          <div style={fieldLabelRowStyle}>
            <label style={fieldLabelStyle}>Sequential Copier ("Copy & Next")</label>
            <span style={skillsCountBadgeStyle}>
              {safeIndex + 1} of {skillsCount}
            </span>
          </div>
          <p style={helpTextStyle}>
            Click button to copy current skill and advance to next for rapid pasting into job forms.
          </p>
          <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '10px' }}>
            <button
              type="button"
              onClick={handleCopyCurrentAndAdvance}
              style={{
                ...primaryButtonStyle,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: '8px',
                padding: '11px 16px',
                fontSize: '13px',
                fontWeight: 700,
                backgroundColor: copiedSkill === currentSkill ? '#059669' : '#006699',
              }}
            >
              {copiedSkill === currentSkill ? (
                <IconCheck size={15} color="#FFFFFF" />
              ) : (
                <IconCopy size={15} />
              )}
              <span>
                {copiedSkill === currentSkill
                  ? `Copied "${currentSkill}"!`
                  : `Copy "${currentSkill}" & Next →`}
              </span>
            </button>

            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '8px' }}>
              <button
                type="button"
                onClick={handleStepPrev}
                style={secondaryButtonStyle}
                title="Previous skill"
              >
                ← Prev
              </button>
              <button
                type="button"
                onClick={handleResetStepper}
                style={secondaryButtonStyle}
                title="Reset to first skill"
              >
                ↺ Reset
              </button>
              <button
                type="button"
                onClick={handleStepNext}
                style={secondaryButtonStyle}
                title="Skip to next skill"
              >
                Skip Next →
              </button>
            </div>
          </div>
        </div>
      )}

      <div style={cardContainerStyle}>
        <label style={fieldLabelStyle}>Default Injection Strategy</label>
        <div style={segmentedControlContainerStyle}>
          <button
            type="button"
            onClick={() => handleModeChange('cycle_one_by_one')}
            style={{
              ...segmentedControlButtonStyle,
              ...(configuration.skillsDefaultMode === 'cycle_one_by_one'
                ? segmentedControlButtonActiveStyle
                : {}),
            }}
          >
            Cycle One-by-One
          </button>
          <button
            type="button"
            onClick={() => handleModeChange('dump_all')}
            style={{
              ...segmentedControlButtonStyle,
              ...(configuration.skillsDefaultMode === 'dump_all'
                ? segmentedControlButtonActiveStyle
                : {}),
            }}
          >
            Dump All (Comma)
          </button>
        </div>
        <p style={helpTextStyle}>
          {configuration.skillsDefaultMode === 'cycle_one_by_one'
            ? 'Types each skill followed by Enter into tag / combobox inputs.'
            : 'Inserts all skills as a single comma-separated text string.'}
        </p>
      </div>

      <div style={cardContainerStyle}>
        <label style={fieldLabelStyle}>Add Technical Skill</label>
        <div style={addSkillInputRowStyle}>
          <input
            type="text"
            placeholder="e.g. Go, Python, React, PostgreSQL..."
            value={newSkillText}
            onInput={(event) =>
              setNewSkillText((event.target as HTMLInputElement).value)
            }
            onKeyDown={handleKeyDown}
            style={textInputStyle}
          />
          <button
            type="button"
            onClick={handleAddSkill}
            style={primaryButtonStyle}
          >
            Add
          </button>
        </div>
      </div>

      <div style={cardContainerStyle}>
        <div style={fieldLabelRowStyle}>
          <label style={fieldLabelStyle}>Configured Skills</label>
          <span style={skillsCountBadgeStyle}>{skillsCount}</span>
        </div>
        {skillsCount === 0 ? (
          <p style={emptyStateTextStyle}>
            No skills added yet. Add programming languages, frameworks, or tools above.
          </p>
        ) : (
          <div style={chipsContainerStyle}>
            {skills.map((skill, index) => {
              const isSelected = index === safeIndex;
              const isCopied = copiedSkill === skill;
              return (
                <span
                  key={skill}
                  onClick={() => handleCopySpecificSkill(skill, index)}
                  style={{
                    ...skillChipStyle,
                    cursor: 'pointer',
                    borderColor: isCopied ? '#10B981' : isSelected ? '#006699' : '#C6C6CD',
                    backgroundColor: isCopied ? '#ECFDF5' : isSelected ? '#E8F1FF' : '#F7F9FB',
                    color: isCopied ? '#065F46' : isSelected ? '#004B73' : '#191C1E',
                    fontWeight: isSelected ? 700 : 500,
                  }}
                  title={`Click to copy "${skill}"`}
                >
                  <span style={{ display: 'inline-flex', alignItems: 'center', gap: '4px' }}>
                    {isCopied ? (
                      <IconCheck size={11} color="#059669" />
                    ) : (
                      <IconCopy size={11} />
                    )}
                    <span>{skill}</span>
                  </span>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleRemoveSkill(skill);
                    }}
                    style={removeChipButtonStyle}
                    title="Remove skill"
                  >
                    ✕
                  </button>
                </span>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

interface ConnectionSettingsProperties {
  configuration: ExtensionConfig;
  isSyncing: boolean;
  onConfigurationChange: (updatedConfiguration: ExtensionConfig) => void;
  onSyncFromBackend: () => void;
  onNotification: (message: string, type: 'success' | 'error') => void;
  onReloadData: () => Promise<void>;
}

/**
 * Backend connection settings: Option B (Email/Password) + Option C (Web Auth Bridge) + Manual JWT.
 */
function ConnectionSettings({
  configuration,
  isSyncing,
  onConfigurationChange,
  onSyncFromBackend,
  onNotification,
  onReloadData,
}: ConnectionSettingsProperties): JSX.Element {
  const [email, setEmail] = useState<string>('');
  const [password, setPassword] = useState<string>('');
  const [isLoggingIn, setIsLoggingIn] = useState<boolean>(false);
  const [showManualToken, setShowManualToken] = useState<boolean>(false);
  const [showSecretText, setShowSecretText] = useState<boolean>(false);

  const isActuallyConnected =
    configuration.mode === 'connected' && Boolean(configuration.apiSecret?.trim());

  let backendHostname = 'DuckDNS Cloud API';
  try {
    const target = resolveBackendUrl(configuration.backendUrl);
    backendHostname = new URL(target).hostname;
  } catch {
    backendHostname = 'Live API';
  }

  const handleBackendUrlChange = (value: string) => {
    onConfigurationChange({
      ...configuration,
      backendUrl: value,
    });
  };

  const handleApiSecretChange = (value: string) => {
    onConfigurationChange({
      ...configuration,
      apiSecret: value,
    });
  };

  const handleDisconnect = () => {
    onConfigurationChange({
      ...configuration,
      mode: 'local',
      apiSecret: '',
    });
    onNotification('Disconnected (switched to Local Vault)', 'success');
  };

  const handleLoginWithCredentials = async () => {
    if (!email.trim() || !password.trim()) {
      onNotification('Please enter both Email and Password', 'error');
      return;
    }

    setIsLoggingIn(true);
    try {
      const response = (await browser.runtime.sendMessage({
        action: 'LOGIN_WITH_CREDENTIALS',
        payload: {
          email: email.trim(),
          password: password,
          backendUrl: configuration.backendUrl.trim() || DEFAULT_BACKEND_URL,
        },
      })) as { success: boolean; error?: string } | undefined;

      if (response && response.success) {
        await onReloadData();
        onNotification('Successfully logged in and synced!', 'success');
        setPassword('');
      } else {
        onNotification(response?.error ?? 'Login failed', 'error');
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Login failed';
      onNotification(msg, 'error');
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleOpenGoogleWebAuth = async () => {
    try {
      const detectResult = (await browser.runtime.sendMessage({
        action: 'DETECT_WEB_SESSION',
      })) as { success?: boolean; error?: string } | undefined;
      if (detectResult?.success) {
        await onReloadData();
        onNotification(
          'Active web session found! Connected and synced automatically.',
          'success',
        );
        return;
      }
    } catch {
      return;
    }

    const trimmedUrl = configuration.backendUrl.trim();
    const targetWebUrl = trimmedUrl.includes('web.app')
      ? trimmedUrl
      : DEFAULT_WEB_PORTAL_URL;

    void browser.tabs.create({ url: targetWebUrl });
    onNotification(
      'Opening Job Cruiser Web. Sign in with Google — Quick Fill connects automatically!',
      'success',
    );
  };

  const formattedLastSynced = configuration.lastSyncedAt
    ? new Date(configuration.lastSyncedAt).toLocaleTimeString()
    : 'Never';

  return (
    <div style={settingsContainerStyle}>
      <div style={statusBannerCardStyle}>
        <div style={statusRowStyle}>
          <span style={statusTitleStyle}>Status</span>
          <span
            style={{
              ...statusBadgeStyle,
              backgroundColor: isActuallyConnected ? '#10B98115' : '#F2F4F6',
              color: isActuallyConnected ? '#059669' : '#515F74',
              borderColor: isActuallyConnected ? '#10B98140' : '#C6C6CD',
            }}
          >
            {isActuallyConnected
              ? `Connected to ${backendHostname}`
              : 'Local Vault (Not Connected)'}
          </span>
        </div>
      </div>

      {isActuallyConnected ? (
        <div style={cardContainerStyle}>
          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div>
              <div
                style={{
                  fontWeight: 700,
                  fontSize: '13px',
                  color: '#191C1E',
                }}
              >
                Cloud Sync Active
              </div>
              <div style={helpTextStyle}>
                Token: ••••••••{configuration.apiSecret.slice(-6)}
              </div>
              <div style={helpTextStyle}>
                Last synced: {formattedLastSynced}
              </div>
            </div>
            <div style={{ display: 'flex', gap: '8px', alignItems: 'center' }}>
              <button
                type="button"
                onClick={onSyncFromBackend}
                disabled={isSyncing}
                style={secondaryButtonStyle}
              >
                <IconRotateCw size={13} isSpinning={isSyncing} />
                <span>Sync</span>
              </button>
              <button
                type="button"
                onClick={handleDisconnect}
                style={disconnectButtonStyle}
              >
                Disconnect
              </button>
            </div>
          </div>
        </div>
      ) : (
        <>
          <div style={cardContainerStyle}>
            <label style={fieldLabelStyle}>Google Single Sign-On</label>
            <p style={helpTextStyle}>
              Sign in on your Job Cruiser web portal. Quick Fill will automatically detect your session and sync your vault.
            </p>

            <button
              type="button"
              onClick={handleOpenGoogleWebAuth}
              style={{ ...googleButtonStyle, marginTop: '8px' }}
            >
              <IconGoogle size={15} />
              <span>Sign in with Google (Web Portal)</span>
            </button>
          </div>

          <div style={cardContainerStyle}>
            <label style={fieldLabelStyle}>Direct Account Login</label>
            <p style={helpTextStyle}>
              Sign in directly with your registered Job Cruiser email and password.
            </p>

            <div
              style={{
                display: 'flex',
                flexDirection: 'column',
                gap: '8px',
                marginTop: '8px',
              }}
            >
              <input
                type="email"
                placeholder="name@example.com"
                value={email}
                onInput={(e) =>
                  setEmail((e.target as HTMLInputElement).value)
                }
                style={textInputStyle}
              />
              <input
                type="password"
                placeholder="Password"
                value={password}
                onInput={(e) =>
                  setPassword((e.target as HTMLInputElement).value)
                }
                style={textInputStyle}
              />
              <button
                type="button"
                onClick={handleLoginWithCredentials}
                disabled={isLoggingIn}
                style={primaryButtonStyle}
              >
                {isLoggingIn ? 'Logging in...' : 'Log In & Sync Vault'}
              </button>
            </div>
          </div>
        </>
      )}

      <div style={cardContainerStyle}>
        <label style={fieldLabelStyle}>Job Cruiser Backend URL</label>
        <input
          type="text"
          value={configuration.backendUrl}
          onInput={(e) =>
            handleBackendUrlChange((e.target as HTMLInputElement).value)
          }
          placeholder="https://job-cruiser-backend.duckdns.org"
          style={textInputStyle}
        />
        <p style={helpTextStyle}>
          Backend API endpoint. If you enter divine-beanbag-468110-v1.web.app, Quick Fill automatically connects to the backend API.
        </p>
      </div>

      {!isActuallyConnected && (
        <div style={{ marginTop: '4px' }}>
          <button
            type="button"
            onClick={() => setShowManualToken(!showManualToken)}
            style={linkToggleButtonStyle}
          >
            {showManualToken
              ? 'Hide manual token'
              : 'Advanced: Paste Bearer Token manually'}
          </button>

          {showManualToken && (
            <div style={{ ...cardContainerStyle, marginTop: '8px' }}>
              <label style={fieldLabelStyle}>Manual Bearer Token</label>
              <div style={passwordInputContainerStyle}>
                <input
                  type={showSecretText ? 'text' : 'password'}
                  value={configuration.apiSecret}
                  onInput={(e) =>
                    handleApiSecretChange((e.target as HTMLInputElement).value)
                  }
                  placeholder="Paste JWT token string"
                  style={textInputStyle}
                />
                <button
                  type="button"
                  onClick={() => setShowSecretText(!showSecretText)}
                  style={passwordVisibilityButtonStyle}
                >
                  {showSecretText ? 'Hide' : 'Show'}
                </button>
              </div>
              <button
                type="button"
                disabled={isSyncing}
                onClick={async () => {
                  onConfigurationChange({
                    ...configuration,
                    mode: 'connected',
                  });
                  await onSyncFromBackend();
                }}
                style={{
                  ...secondaryButtonStyle,
                  marginTop: '8px',
                  width: '100%',
                }}
              >
                {isSyncing ? 'Connecting & Syncing...' : 'Connect with Token'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}


/**
 * Styles (strictly matching Job Cruiser Flutter AppColors & Theme)
 */
const containerStyle: JSX.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  minHeight: '100vh',
  backgroundColor: '#F7F9FB',
  color: '#191C1E',
  fontFamily: "'Inter', system-ui, -apple-system, sans-serif",
  width: '100%',
  margin: '0 auto',
};

const headerStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  borderBottom: '1px solid #E2E8F0',
  padding: '12px 14px 0 14px',
  position: 'sticky',
  top: 0,
  zIndex: 10,
};

const headerTitleRowStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  marginBottom: '10px',
};

const brandTitleStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '10px',
};

const boltContainerStyle: JSX.CSSProperties = {
  width: '30px',
  height: '30px',
  borderRadius: '8px',
  backgroundColor: '#F2F4F6',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const brandTextStyle: JSX.CSSProperties = {
  fontWeight: 800,
  fontSize: '14px',
  color: '#191C1E',
  letterSpacing: '-0.02em',
  lineHeight: 1.1,
};

const brandSubtitleStyle: JSX.CSSProperties = {
  fontSize: '11px',
  color: '#515F74',
  fontWeight: 500,
};

const statusBadgeStyle: JSX.CSSProperties = {
  fontSize: '11px',
  padding: '2px 8px',
  borderRadius: '12px',
  border: '1px solid',
  fontWeight: 600,
};

const iconRefreshButtonStyle: JSX.CSSProperties = {
  background: '#F2F4F6',
  border: '1px solid #E2E8F0',
  borderRadius: '6px',
  color: '#191C1E',
  cursor: 'pointer',
  padding: '5px 7px',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const tabNavigationStyle: JSX.CSSProperties = {
  display: 'flex',
  gap: '4px',
};

const tabButtonStyle: JSX.CSSProperties = {
  flex: 1,
  padding: '9px 0',
  backgroundColor: 'transparent',
  border: 'none',
  borderBottom: '2px solid transparent',
  color: '#76777D',
  cursor: 'pointer',
  fontWeight: 600,
  fontSize: '12px',
  textAlign: 'center',
};

const activeTabButtonStyle: JSX.CSSProperties = {
  color: '#000000',
  borderBottom: '2px solid #000000',
};

const notificationBannerStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  padding: '9px 12px',
  margin: '8px 12px 0 12px',
  borderRadius: '8px',
  border: '1px solid',
  fontSize: '12px',
};

const notificationCloseButtonStyle: JSX.CSSProperties = {
  background: 'none',
  border: 'none',
  color: 'inherit',
  cursor: 'pointer',
  fontSize: '12px',
  padding: 0,
};

const mainContentStyle: JSX.CSSProperties = {
  flex: 1,
  padding: '12px',
  overflowY: 'auto',
};

const vaultContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '10px',
};

const topActionBarStyle: JSX.CSSProperties = {
  display: 'flex',
  gap: '8px',
  marginBottom: '2px',
};

const primaryButtonStyle: JSX.CSSProperties = {
  backgroundColor: '#000000',
  color: '#FFFFFF',
  border: 'none',
  borderRadius: '8px',
  padding: '9px 16px',
  fontWeight: 600,
  fontSize: '12px',
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: '100%',
};

const secondaryButtonStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  color: '#191C1E',
  border: '1px solid #C6C6CD',
  borderRadius: '8px',
  padding: '8px 12px',
  fontWeight: 600,
  fontSize: '12px',
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: '6px',
  lineHeight: 1,
};

const googleButtonStyle: JSX.CSSProperties = {
  width: '100%',
  backgroundColor: '#FFFFFF',
  color: '#191C1E',
  border: '1px solid #C6C6CD',
  borderRadius: '8px',
  padding: '9px 14px',
  fontWeight: 600,
  fontSize: '12px',
  cursor: 'pointer',
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: '8px',
  marginTop: '8px',
};

const disconnectButtonStyle: JSX.CSSProperties = {
  backgroundColor: '#FEE2E2',
  color: '#BA1A1A',
  border: '1px solid #FECACA',
  borderRadius: '6px',
  padding: '5px 10px',
  fontWeight: 600,
  fontSize: '11px',
  cursor: 'pointer',
};

const deleteIconButtonStyle: JSX.CSSProperties = {
  background: '#FEE2E2',
  border: '1px solid #FECACA',
  borderRadius: '6px',
  color: '#BA1A1A',
  cursor: 'pointer',
  padding: '4px 6px',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const textInputStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  border: '1px solid #C6C6CD',
  borderRadius: '8px',
  padding: '8px 10px',
  color: '#191C1E',
  fontSize: '12px',
  outline: 'none',
  width: '100%',
  boxSizing: 'border-box',
};

const textareaInputStyle: JSX.CSSProperties = {
  ...textInputStyle,
  minHeight: '70px',
  resize: 'vertical',
};

const quickFillCardContainerStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  borderRadius: '12px',
  border: '1px solid #E2E8F0',
  padding: '12px 14px',
  boxShadow: '0 1px 3px rgba(0, 0, 0, 0.03)',
  cursor: 'pointer',
  display: 'flex',
  flexDirection: 'column',
  gap: '8px',
  transition: 'border-color 0.15s ease, box-shadow 0.15s ease',
};

const quickFillCardEditingContainerStyle: JSX.CSSProperties = {
  ...quickFillCardContainerStyle,
  border: '1.5px solid #000000',
  cursor: 'default',
  boxShadow: '0 2px 6px rgba(0, 0, 0, 0.08)',
};

const quickFillCardHeaderRowStyle: JSX.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
};

const quickFillCardLabelBoxStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '8px',
  flex: 1,
  overflow: 'hidden',
};

const quickFillCardLabelTextStyle: JSX.CSSProperties = {
  fontSize: '13px',
  fontWeight: 700,
  color: '#191C1E',
  overflow: 'hidden',
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
};

const quickFillCardCategoryBadgeStyle: JSX.CSSProperties = {
  fontSize: '10px',
  fontWeight: 600,
  color: '#515F74',
  backgroundColor: '#F2F4F6',
  padding: '1px 6px',
  borderRadius: '6px',
  whiteSpace: 'nowrap',
};

const quickFillCardActionButtonsBoxStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: '6px',
};

const quickFillIconActionButtonStyle: JSX.CSSProperties = {
  background: '#F2F4F6',
  border: '1px solid #E2E8F0',
  borderRadius: '6px',
  color: '#515F74',
  cursor: 'pointer',
  padding: '4px 6px',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
};

const quickFillCopyBadgeStyle: JSX.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '4px',
  padding: '3px 8px',
  borderRadius: '6px',
  fontSize: '11px',
  fontWeight: 700,
  cursor: 'pointer',
  transition: 'background-color 0.15s ease, color 0.15s ease',
};

const quickFillCardValueBoxStyle: JSX.CSSProperties = {
  backgroundColor: '#F2F4F6',
  border: '1px solid #E2E8F0',
  borderRadius: '8px',
  padding: '8px 10px',
  minHeight: '34px',
  boxSizing: 'border-box',
  display: 'flex',
  alignItems: 'center',
};

const quickFillCardValueTextStyle: JSX.CSSProperties = {
  fontFamily: "'SFMono-Regular', Consolas, Menlo, monospace",
  fontSize: '12px',
  color: '#191C1E',
  wordBreak: 'break-word',
  lineHeight: 1.4,
};

const expandToggleButtonStyle: JSX.CSSProperties = {
  backgroundColor: '#E2E8F0',
  border: 'none',
  borderRadius: '4px',
  color: '#006699',
  fontSize: '11px',
  fontWeight: 700,
  padding: '2px 6px',
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  gap: '3px',
  flexShrink: 0,
  marginLeft: '6px',
};

const quickFillCardEditActionsRowStyle: JSX.CSSProperties = {
  display: 'flex',
  justifyContent: 'flex-end',
  gap: '8px',
  marginTop: '8px',
};

const compactPrimaryButtonStyle: JSX.CSSProperties = {
  ...primaryButtonStyle,
  width: 'auto',
  padding: '6px 14px',
  fontSize: '11px',
};

const disconnectedBannerCardStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  border: '1px solid #E2E8F0',
  borderRadius: '10px',
  padding: '10px 12px',
  display: 'flex',
  flexDirection: 'column',
  gap: '6px',
  marginBottom: '4px',
};

const topActionBarRowStyle: JSX.CSSProperties = {
  display: 'flex',
  gap: '8px',
  alignItems: 'center',
};

const copyEverythingButtonStyle: JSX.CSSProperties = {
  backgroundColor: '#000000',
  color: '#FFFFFF',
  border: 'none',
  borderRadius: '8px',
  padding: '9px 14px',
  fontSize: '12px',
  fontWeight: 700,
  cursor: 'pointer',
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: '6px',
  flex: 1,
  transition: 'background-color 0.2s ease',
};

const quickFillSearchContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  position: 'relative',
  width: '100%',
};

const quickFillSearchInputStyle: JSX.CSSProperties = {
  ...textInputStyle,
  paddingLeft: '32px',
  paddingRight: '28px',
};

const quickFillSearchIconStyle: JSX.CSSProperties = {
  position: 'absolute',
  left: '10px',
  display: 'flex',
  alignItems: 'center',
  color: '#515F74',
  pointerEvents: 'none',
};

const quickFillClearSearchButtonStyle: JSX.CSSProperties = {
  position: 'absolute',
  right: '8px',
  background: 'none',
  border: 'none',
  color: '#515F74',
  cursor: 'pointer',
  padding: '2px 4px',
  fontSize: '12px',
};

const quickFillCategoryChipsContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  gap: '6px',
  overflowX: 'auto',
  paddingBottom: '4px',
  scrollbarWidth: 'none',
};

const quickFillCategoryChipStyle: JSX.CSSProperties = {
  border: '1px solid #E2E8F0',
  borderRadius: '20px',
  padding: '4px 10px',
  fontSize: '11px',
  fontWeight: 600,
  cursor: 'pointer',
  whiteSpace: 'nowrap',
  transition: 'all 0.15s ease',
};

const fieldLabelStyle: JSX.CSSProperties = {
  fontSize: '11px',
  fontWeight: 600,
  color: '#45464D',
};

const fieldLabelRowStyle: JSX.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
};

const compactInputStyle: JSX.CSSProperties = {
  ...textInputStyle,
  flex: 1,
  padding: '6px 8px',
};

const skillsContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '10px',
};

const cardContainerStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  borderRadius: '10px',
  border: '1px solid #E2E8F0',
  boxShadow: '0 1px 2px rgba(0, 0, 0, 0.03)',
  padding: '12px 14px',
  display: 'flex',
  flexDirection: 'column',
  gap: '8px',
};

const segmentedControlContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  backgroundColor: '#F2F4F6',
  borderRadius: '8px',
  padding: '2px',
  border: '1px solid #E2E8F0',
};

const segmentedControlButtonStyle: JSX.CSSProperties = {
  flex: 1,
  padding: '6px 0',
  backgroundColor: 'transparent',
  border: 'none',
  borderRadius: '6px',
  color: '#515F74',
  fontSize: '11px',
  fontWeight: 600,
  cursor: 'pointer',
  textAlign: 'center',
};

const segmentedControlButtonActiveStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  color: '#191C1E',
  boxShadow: '0 1px 2px rgba(0, 0, 0, 0.06)',
};

const addSkillInputRowStyle: JSX.CSSProperties = {
  display: 'flex',
  gap: '6px',
};

const skillsCountBadgeStyle: JSX.CSSProperties = {
  fontSize: '10px',
  backgroundColor: '#F2F4F6',
  color: '#191C1E',
  padding: '1px 6px',
  borderRadius: '8px',
  fontWeight: 700,
};

const chipsContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  flexWrap: 'wrap',
  gap: '6px',
  maxHeight: '200px',
  overflowY: 'auto',
};

const skillChipStyle: JSX.CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  gap: '6px',
  backgroundColor: '#F2F4F6',
  border: '1px solid #E2E8F0',
  borderRadius: '6px',
  padding: '4px 9px',
  fontSize: '11px',
  color: '#191C1E',
  fontWeight: 600,
};

const removeChipButtonStyle: JSX.CSSProperties = {
  background: 'none',
  border: 'none',
  color: '#76777D',
  cursor: 'pointer',
  padding: 0,
  fontSize: '10px',
  display: 'inline-flex',
};

const emptyStateTextStyle: JSX.CSSProperties = {
  fontSize: '11px',
  color: '#76777D',
  margin: '4px 0',
};

const settingsContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  flexDirection: 'column',
  gap: '10px',
};

const statusBannerCardStyle: JSX.CSSProperties = {
  backgroundColor: '#FFFFFF',
  borderRadius: '10px',
  border: '1px solid #E2E8F0',
  boxShadow: '0 1px 2px rgba(0, 0, 0, 0.03)',
  padding: '12px 14px',
};

const statusRowStyle: JSX.CSSProperties = {
  display: 'flex',
  justifyContent: 'space-between',
  alignItems: 'center',
};

const statusTitleStyle: JSX.CSSProperties = {
  fontWeight: 700,
  fontSize: '13px',
  color: '#191C1E',
};

const helpTextStyle: JSX.CSSProperties = {
  fontSize: '11px',
  color: '#76777D',
  margin: 0,
};

const passwordInputContainerStyle: JSX.CSSProperties = {
  display: 'flex',
  position: 'relative',
  alignItems: 'center',
};

const passwordVisibilityButtonStyle: JSX.CSSProperties = {
  position: 'absolute',
  right: '8px',
  background: 'none',
  border: 'none',
  color: '#000000',
  fontSize: '11px',
  cursor: 'pointer',
  fontWeight: 600,
};

const linkToggleButtonStyle: JSX.CSSProperties = {
  background: 'none',
  border: 'none',
  color: '#515F74',
  fontSize: '11px',
  fontWeight: 600,
  cursor: 'pointer',
  padding: '4px 0',
  textAlign: 'left',
};

/**
 * Mount Preact application into sidepanel document root.
 */
if (typeof document !== 'undefined') {
  const rootElement = document.getElementById('root');
  if (rootElement) {
    render(<App />, rootElement);
  }
}
