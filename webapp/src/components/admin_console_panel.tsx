import React, {useEffect, useState} from 'react';

import manifest from 'plugin_manifest';

type PreviewResponse = {
    user_id: string;
    username: string;
    language: string;
    department_code: string;
    department_name: string;
    common_template_id: string;
    department_template_id?: string;
    applied_template_ids: string[];
    message: string;
    skipped_link_count: number;
};

type SendLog = {
    log_id: string;
    user_id: string;
    username: string;
    dept_code?: string;
    template_ids: string[];
    sent_at: number;
    status: string;
    error_message?: string;
    mode: string;
    requested_by?: string;
};

type StatsResponse = {
    total_sends: number;
    successful: number;
    failed: number;
    skipped: number;
    manual_resends: number;
    by_department: Record<string, number>;
    recent_failures: string[];
};

type FormState = {
    userId: string;
    username: string;
    departmentCode: string;
    departmentName: string;
    organizationName: string;
    language: string;
    startDate: string;
};

type AdminConfig = {
    enable_auto_send: boolean;
    sender_bot_username: string;
    sender_bot_display_name: string;
    default_language: string;
    fallback_department_code: string;
    initial_delay_seconds: number;
    retry_interval_minutes: number;
    retry_max_attempts: number;
    templates_json: string;
    links_json: string;
    department_mappings_json: string;
    exclusion_rules_json: string;
};

const pluginBasePath = `/plugins/${manifest.id}/api/v1`;

const defaultFormState: FormState = {
    userId: '',
    username: '',
    departmentCode: '',
    departmentName: '',
    organizationName: '',
    language: '',
    startDate: '',
};

const defaultConfigState: AdminConfig = {
    enable_auto_send: true,
    sender_bot_username: 'onboarding.bot',
    sender_bot_display_name: 'Onboarding Bot',
    default_language: 'ko',
    fallback_department_code: 'DEFAULT',
    initial_delay_seconds: 0,
    retry_interval_minutes: 5,
    retry_max_attempts: 3,
    templates_json: '',
    links_json: '',
    department_mappings_json: '[]',
    exclusion_rules_json: '',
};

const styles = {
    shell: {
        display: 'grid',
        gap: '16px',
    } as React.CSSProperties,
    card: {
        border: '1px solid rgba(61, 60, 64, 0.12)',
        borderRadius: '12px',
        padding: '16px',
        background: '#ffffff',
        boxShadow: '0 8px 24px rgba(23, 43, 77, 0.04)',
    } as React.CSSProperties,
    cardTitle: {
        margin: 0,
        fontSize: '16px',
        fontWeight: 600,
        color: '#1f2329',
    } as React.CSSProperties,
    cardSubtitle: {
        margin: '6px 0 0',
        color: '#5d6677',
        fontSize: '13px',
        lineHeight: 1.5,
    } as React.CSSProperties,
    sectionTitle: {
        margin: '18px 0 0',
        fontSize: '13px',
        fontWeight: 700,
        color: '#394150',
        textTransform: 'uppercase',
        letterSpacing: '0.04em',
    } as React.CSSProperties,
    grid: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
        gap: '12px',
        marginTop: '16px',
    } as React.CSSProperties,
    jsonGrid: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(320px, 1fr))',
        gap: '12px',
        marginTop: '16px',
    } as React.CSSProperties,
    field: {
        display: 'grid',
        gap: '6px',
    } as React.CSSProperties,
    toggleField: {
        display: 'grid',
        gap: '8px',
        padding: '12px',
        borderRadius: '10px',
        border: '1px solid #dfe6f1',
        background: '#f8fbff',
    } as React.CSSProperties,
    toggleLabel: {
        display: 'flex',
        alignItems: 'center',
        gap: '10px',
        fontSize: '14px',
        fontWeight: 600,
        color: '#1f2329',
    } as React.CSSProperties,
    helpText: {
        color: '#5d6677',
        fontSize: '12px',
        lineHeight: 1.5,
    } as React.CSSProperties,
    label: {
        fontSize: '12px',
        fontWeight: 600,
        color: '#394150',
    } as React.CSSProperties,
    input: {
        width: '100%',
        minHeight: '40px',
        padding: '10px 12px',
        borderRadius: '8px',
        border: '1px solid #c5cedd',
        fontSize: '14px',
        color: '#1f2329',
        background: '#ffffff',
        boxSizing: 'border-box',
    } as React.CSSProperties,
    textarea: {
        width: '100%',
        minHeight: '220px',
        padding: '12px',
        borderRadius: '8px',
        border: '1px solid #c5cedd',
        fontSize: '14px',
        lineHeight: 1.6,
        color: '#1f2329',
        background: '#fbfcfe',
        boxSizing: 'border-box',
        whiteSpace: 'pre-wrap',
    } as React.CSSProperties,
    codeArea: {
        width: '100%',
        minHeight: '280px',
        padding: '12px',
        borderRadius: '8px',
        border: '1px solid #c5cedd',
        fontSize: '13px',
        lineHeight: 1.6,
        color: '#1f2329',
        background: '#f7f9fc',
        boxSizing: 'border-box',
        fontFamily: 'Consolas, Monaco, monospace',
    } as React.CSSProperties,
    buttonRow: {
        display: 'flex',
        flexWrap: 'wrap',
        gap: '10px',
        marginTop: '16px',
    } as React.CSSProperties,
    buttonPrimary: {
        minHeight: '40px',
        padding: '0 16px',
        borderRadius: '999px',
        border: 'none',
        background: '#0070f3',
        color: '#ffffff',
        fontWeight: 600,
        cursor: 'pointer',
    } as React.CSSProperties,
    buttonSecondary: {
        minHeight: '40px',
        padding: '0 16px',
        borderRadius: '999px',
        border: '1px solid #c5cedd',
        background: '#ffffff',
        color: '#1f2329',
        fontWeight: 600,
        cursor: 'pointer',
    } as React.CSSProperties,
    badgeRow: {
        display: 'flex',
        flexWrap: 'wrap',
        gap: '8px',
        marginTop: '14px',
    } as React.CSSProperties,
    badge: {
        display: 'inline-flex',
        alignItems: 'center',
        gap: '6px',
        padding: '6px 10px',
        borderRadius: '999px',
        background: '#eef4ff',
        color: '#184f99',
        fontSize: '12px',
        fontWeight: 600,
    } as React.CSSProperties,
    status: {
        marginTop: '12px',
        fontSize: '13px',
        lineHeight: 1.5,
    } as React.CSSProperties,
    statsGrid: {
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))',
        gap: '12px',
        marginTop: '16px',
    } as React.CSSProperties,
    statBox: {
        borderRadius: '10px',
        background: 'linear-gradient(145deg, #f7f9fc 0%, #eef4ff 100%)',
        padding: '14px',
    } as React.CSSProperties,
    statLabel: {
        fontSize: '12px',
        color: '#5d6677',
    } as React.CSSProperties,
    statValue: {
        marginTop: '6px',
        fontSize: '24px',
        fontWeight: 700,
        color: '#172b4d',
    } as React.CSSProperties,
    table: {
        width: '100%',
        borderCollapse: 'collapse',
        marginTop: '16px',
    } as React.CSSProperties,
    th: {
        textAlign: 'left',
        padding: '10px 8px',
        fontSize: '12px',
        color: '#5d6677',
        borderBottom: '1px solid #dfe6f1',
    } as React.CSSProperties,
    td: {
        padding: '12px 8px',
        fontSize: '13px',
        color: '#1f2329',
        borderBottom: '1px solid #edf1f7',
        verticalAlign: 'top',
    } as React.CSSProperties,
    list: {
        margin: '12px 0 0',
        paddingLeft: '18px',
        color: '#394150',
        fontSize: '13px',
        lineHeight: 1.6,
    } as React.CSSProperties,
    empty: {
        marginTop: '12px',
        color: '#6b7280',
        fontSize: '13px',
    } as React.CSSProperties,
};

async function apiRequest<T>(path: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${pluginBasePath}${path}`, {
        credentials: 'same-origin',
        headers: {
            'Content-Type': 'application/json',
            'X-Requested-With': 'XMLHttpRequest',
            ...(options?.headers || {}),
        },
        ...options,
    });

    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
        const message = typeof payload?.error === 'string' ? payload.error : 'Request failed';
        throw new Error(message);
    }

    return payload as T;
}

function formatDate(timestamp: number): string {
    if (!timestamp) {
        return '-';
    }

    return new Date(timestamp).toLocaleString();
}

function formatJSONBlock(value: string): string {
    if (!value.trim()) {
        return '';
    }

    return JSON.stringify(JSON.parse(value), null, 2);
}

export default function AdminConsolePanel() {
    const [form, setForm] = useState<FormState>(defaultFormState);
    const [config, setConfig] = useState<AdminConfig>(defaultConfigState);
    const [preview, setPreview] = useState<PreviewResponse | null>(null);
    const [stats, setStats] = useState<StatsResponse | null>(null);
    const [logs, setLogs] = useState<SendLog[]>([]);
    const [statusMessage, setStatusMessage] = useState<string>('');
    const [errorMessage, setErrorMessage] = useState<string>('');
    const [busy, setBusy] = useState<boolean>(false);
    const [statsBusy, setStatsBusy] = useState<boolean>(false);
    const [configBusy, setConfigBusy] = useState<boolean>(false);
    const [saveBusy, setSaveBusy] = useState<boolean>(false);

    useEffect(() => {
        void Promise.all([
            loadConfiguration(false),
            refreshDashboard(false),
        ]);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const departmentRows = stats ? Object.entries(stats.by_department || {}).sort((a, b) => b[1] - a[1]) : [];

    const updateField = (key: keyof FormState, value: string) => {
        setForm((current) => ({
            ...current,
            [key]: value,
        }));
    };

    const updateConfigField = <K extends keyof AdminConfig>(key: K, value: AdminConfig[K]) => {
        setConfig((current) => ({
            ...current,
            [key]: value,
        }));
    };

    const loadConfiguration = async (announce = true) => {
        setConfigBusy(true);
        if (announce) {
            setStatusMessage('');
            setErrorMessage('');
        }

        try {
            const response = await apiRequest<AdminConfig>('/admin/config');
            setConfig(response);
            if (announce) {
                setStatusMessage('Configuration loaded.');
            }
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to load configuration.');
        } finally {
            setConfigBusy(false);
        }
    };

    const refreshDashboard = async (announce = false) => {
        setStatsBusy(true);
        if (announce) {
            setStatusMessage('');
            setErrorMessage('');
        }

        try {
            const [statsResponse, logsResponse] = await Promise.all([
                apiRequest<StatsResponse>('/admin/stats'),
                apiRequest<SendLog[]>('/admin/logs'),
            ]);

            setStats(statsResponse);
            setLogs(logsResponse);
            if (announce) {
                setStatusMessage('Delivery dashboard refreshed.');
            }
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to load dashboard data.');
        } finally {
            setStatsBusy(false);
        }
    };

    const handleFormatJSON = () => {
        try {
            setConfig((current) => ({
                ...current,
                templates_json: formatJSONBlock(current.templates_json),
                links_json: formatJSONBlock(current.links_json),
                department_mappings_json: formatJSONBlock(current.department_mappings_json),
                exclusion_rules_json: formatJSONBlock(current.exclusion_rules_json),
            }));
            setStatusMessage('JSON blocks formatted.');
            setErrorMessage('');
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to format JSON blocks.');
        }
    };

    const handleSaveConfig = async () => {
        setSaveBusy(true);
        setStatusMessage('');
        setErrorMessage('');

        try {
            const normalizedConfig: AdminConfig = {
                ...config,
                templates_json: formatJSONBlock(config.templates_json),
                links_json: formatJSONBlock(config.links_json),
                department_mappings_json: formatJSONBlock(config.department_mappings_json),
                exclusion_rules_json: formatJSONBlock(config.exclusion_rules_json),
            };

            const savedConfig = await apiRequest<AdminConfig>('/admin/config', {
                method: 'PUT',
                body: JSON.stringify(normalizedConfig),
            });

            setConfig(savedConfig);
            setStatusMessage('Configuration saved and applied.');
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to save configuration.');
        } finally {
            setSaveBusy(false);
        }
    };

    const handlePreview = async () => {
        if (!form.userId && !form.username) {
            setErrorMessage('Enter a user id or username before previewing.');
            return;
        }

        setBusy(true);
        setStatusMessage('');
        setErrorMessage('');

        try {
            const params = new URLSearchParams();
            if (form.userId) {
                params.set('user_id', form.userId);
            }
            if (form.username) {
                params.set('username', form.username);
            }
            if (form.departmentCode) {
                params.set('dept_code', form.departmentCode);
            }
            if (form.departmentName) {
                params.set('dept_name', form.departmentName);
            }
            if (form.organizationName) {
                params.set('organization_name', form.organizationName);
            }
            if (form.language) {
                params.set('lang', form.language);
            }
            if (form.startDate) {
                params.set('start_date', form.startDate);
            }

            const previewResponse = await apiRequest<PreviewResponse>(`/admin/preview?${params.toString()}`);
            setPreview(previewResponse);
            setStatusMessage('Preview loaded.');
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to load preview.');
        } finally {
            setBusy(false);
        }
    };

    const handleResend = async () => {
        if (!form.userId && !form.username) {
            setErrorMessage('Enter a user id or username before resending.');
            return;
        }

        setBusy(true);
        setStatusMessage('');
        setErrorMessage('');

        try {
            const log = await apiRequest<SendLog>('/admin/resend', {
                method: 'POST',
                body: JSON.stringify({
                    user_id: form.userId,
                    username: form.username,
                }),
            });

            setStatusMessage(`Manual resend finished with status: ${log.status}`);
            await refreshDashboard();
        } catch (error) {
            setErrorMessage(error instanceof Error ? error.message : 'Failed to resend onboarding message.');
        } finally {
            setBusy(false);
        }
    };

    return (
        <div style={styles.shell}>
            <section style={styles.card}>
                <h3 style={styles.cardTitle}>Template configuration</h3>
                <p style={styles.cardSubtitle}>
                    Edit the persisted plugin settings here. The same values still back the standard Mattermost plugin configuration keys,
                    but operators can now review and save them from one panel.
                </p>

                <div style={styles.sectionTitle}>Delivery behavior</div>
                <div style={styles.grid}>
                    <label style={styles.toggleField}>
                        <span style={styles.toggleLabel}>
                            <input
                                checked={config.enable_auto_send}
                                onChange={(e) => updateConfigField('enable_auto_send', e.target.checked)}
                                type='checkbox'
                            />
                            Enable automatic onboarding delivery
                        </span>
                        <span style={styles.helpText}>Starts delivery when the user creation event is received.</span>
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Sender bot username</span>
                        <input
                            style={styles.input}
                            value={config.sender_bot_username}
                            onChange={(e) => updateConfigField('sender_bot_username', e.target.value)}
                            placeholder='onboarding.bot'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Sender bot display name</span>
                        <input
                            style={styles.input}
                            value={config.sender_bot_display_name}
                            onChange={(e) => updateConfigField('sender_bot_display_name', e.target.value)}
                            placeholder='Onboarding Bot'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Default language</span>
                        <input
                            style={styles.input}
                            value={config.default_language}
                            onChange={(e) => updateConfigField('default_language', e.target.value)}
                            placeholder='ko'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Fallback department code</span>
                        <input
                            style={styles.input}
                            value={config.fallback_department_code}
                            onChange={(e) => updateConfigField('fallback_department_code', e.target.value)}
                            placeholder='DEFAULT'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Initial delay seconds</span>
                        <input
                            style={styles.input}
                            type='number'
                            value={config.initial_delay_seconds}
                            onChange={(e) => updateConfigField('initial_delay_seconds', Number.parseInt(e.target.value || '0', 10) || 0)}
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Retry interval minutes</span>
                        <input
                            style={styles.input}
                            type='number'
                            value={config.retry_interval_minutes}
                            onChange={(e) => updateConfigField('retry_interval_minutes', Number.parseInt(e.target.value || '0', 10) || 0)}
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Retry max attempts</span>
                        <input
                            style={styles.input}
                            type='number'
                            value={config.retry_max_attempts}
                            onChange={(e) => updateConfigField('retry_max_attempts', Number.parseInt(e.target.value || '0', 10) || 0)}
                        />
                    </label>
                </div>

                <div style={styles.sectionTitle}>JSON blocks</div>
                <div style={styles.jsonGrid}>
                    <label style={styles.field}>
                        <span style={styles.label}>Templates JSON</span>
                        <textarea
                            style={styles.codeArea}
                            value={config.templates_json}
                            onChange={(e) => updateConfigField('templates_json', e.target.value)}
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Links JSON</span>
                        <textarea
                            style={styles.codeArea}
                            value={config.links_json}
                            onChange={(e) => updateConfigField('links_json', e.target.value)}
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Department mappings JSON</span>
                        <textarea
                            style={styles.codeArea}
                            value={config.department_mappings_json}
                            onChange={(e) => updateConfigField('department_mappings_json', e.target.value)}
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Exclusion rules JSON</span>
                        <textarea
                            style={styles.codeArea}
                            value={config.exclusion_rules_json}
                            onChange={(e) => updateConfigField('exclusion_rules_json', e.target.value)}
                        />
                    </label>
                </div>

                <div style={styles.buttonRow}>
                    <button style={styles.buttonSecondary} onClick={() => void loadConfiguration()} disabled={configBusy || saveBusy}>
                        {configBusy ? 'Loading...' : 'Reload config'}
                    </button>
                    <button style={styles.buttonSecondary} onClick={handleFormatJSON} disabled={saveBusy}>
                        Format JSON
                    </button>
                    <button style={styles.buttonPrimary} onClick={handleSaveConfig} disabled={configBusy || saveBusy}>
                        {saveBusy ? 'Saving...' : 'Save configuration'}
                    </button>
                </div>

                {statusMessage && (
                    <div style={{...styles.status, color: '#116149'}}>{statusMessage}</div>
                )}
                {errorMessage && (
                    <div style={{...styles.status, color: '#c53030'}}>{errorMessage}</div>
                )}
            </section>

            <section style={styles.card}>
                <h3 style={styles.cardTitle}>Operations panel</h3>
                <p style={styles.cardSubtitle}>
                    Preview the currently saved configuration against a target user, resend a message manually, and refresh delivery metrics.
                </p>

                <div style={styles.grid}>
                    <label style={styles.field}>
                        <span style={styles.label}>User ID</span>
                        <input
                            style={styles.input}
                            value={form.userId}
                            onChange={(e) => updateField('userId', e.target.value)}
                            placeholder='target-user-id'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Username</span>
                        <input
                            style={styles.input}
                            value={form.username}
                            onChange={(e) => updateField('username', e.target.value)}
                            placeholder='honggildong'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Department code override</span>
                        <input
                            style={styles.input}
                            value={form.departmentCode}
                            onChange={(e) => updateField('departmentCode', e.target.value)}
                            placeholder='IT'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Department name override</span>
                        <input
                            style={styles.input}
                            value={form.departmentName}
                            onChange={(e) => updateField('departmentName', e.target.value)}
                            placeholder='IT Platform'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Organization override</span>
                        <input
                            style={styles.input}
                            value={form.organizationName}
                            onChange={(e) => updateField('organizationName', e.target.value)}
                            placeholder='AI Office'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Language override</span>
                        <input
                            style={styles.input}
                            value={form.language}
                            onChange={(e) => updateField('language', e.target.value)}
                            placeholder='ko'
                        />
                    </label>
                    <label style={styles.field}>
                        <span style={styles.label}>Start date override</span>
                        <input
                            style={styles.input}
                            value={form.startDate}
                            onChange={(e) => updateField('startDate', e.target.value)}
                            placeholder='2026-03-27'
                        />
                    </label>
                </div>

                <div style={styles.buttonRow}>
                    <button style={styles.buttonPrimary} onClick={handlePreview} disabled={busy}>
                        {busy ? 'Working...' : 'Preview'}
                    </button>
                    <button style={styles.buttonSecondary} onClick={handleResend} disabled={busy}>
                        Manual resend
                    </button>
                    <button style={styles.buttonSecondary} onClick={() => void refreshDashboard(true)} disabled={statsBusy}>
                        {statsBusy ? 'Refreshing...' : 'Refresh stats'}
                    </button>
                </div>

            </section>

            <section style={styles.card}>
                <h3 style={styles.cardTitle}>Rendered preview</h3>
                <p style={styles.cardSubtitle}>
                    This renders the exact onboarding message resolved for the selected user and optional overrides.
                </p>

                {preview ? (
                    <>
                        <div style={styles.badgeRow}>
                            <span style={styles.badge}>Language: {preview.language}</span>
                            <span style={styles.badge}>Department: {preview.department_code || 'N/A'}</span>
                            <span style={styles.badge}>Common template: {preview.common_template_id}</span>
                            {preview.department_template_id && (
                                <span style={styles.badge}>Department template: {preview.department_template_id}</span>
                            )}
                        </div>
                        <textarea
                            style={styles.textarea}
                            readOnly={true}
                            value={preview.message}
                        />
                    </>
                ) : (
                    <div style={styles.empty}>No preview has been loaded yet.</div>
                )}
            </section>

            <section style={styles.card}>
                <h3 style={styles.cardTitle}>Delivery stats</h3>
                <p style={styles.cardSubtitle}>
                    Current statistics are aggregated from KV-backed send logs and are ready to be replaced by a dedicated store later.
                </p>

                {stats ? (
                    <>
                        <div style={styles.statsGrid}>
                            <div style={styles.statBox}>
                                <div style={styles.statLabel}>Total sends</div>
                                <div style={styles.statValue}>{stats.total_sends}</div>
                            </div>
                            <div style={styles.statBox}>
                                <div style={styles.statLabel}>Successful</div>
                                <div style={styles.statValue}>{stats.successful}</div>
                            </div>
                            <div style={styles.statBox}>
                                <div style={styles.statLabel}>Failed</div>
                                <div style={styles.statValue}>{stats.failed}</div>
                            </div>
                            <div style={styles.statBox}>
                                <div style={styles.statLabel}>Skipped</div>
                                <div style={styles.statValue}>{stats.skipped}</div>
                            </div>
                            <div style={styles.statBox}>
                                <div style={styles.statLabel}>Manual resends</div>
                                <div style={styles.statValue}>{stats.manual_resends}</div>
                            </div>
                        </div>

                        {departmentRows.length > 0 ? (
                            <table style={styles.table}>
                                <thead>
                                    <tr>
                                        <th style={styles.th}>Department</th>
                                        <th style={styles.th}>Count</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {departmentRows.map(([department, count]) => (
                                        <tr key={department}>
                                            <td style={styles.td}>{department}</td>
                                            <td style={styles.td}>{count}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        ) : (
                            <div style={styles.empty}>No department rollup is available.</div>
                        )}

                        {stats.recent_failures.length > 0 ? (
                            <ul style={styles.list}>
                                {stats.recent_failures.map((failure) => (
                                    <li key={failure}>{failure}</li>
                                ))}
                            </ul>
                        ) : (
                            <div style={styles.empty}>No recent failures were recorded.</div>
                        )}
                    </>
                ) : (
                    <div style={styles.empty}>Statistics are not available yet.</div>
                )}
            </section>

            <section style={styles.card}>
                <h3 style={styles.cardTitle}>Recent send logs</h3>
                <p style={styles.cardSubtitle}>
                    The newest 10 rows from the latest 100 stored send logs are displayed here.
                </p>

                {logs.length > 0 ? (
                    <table style={styles.table}>
                        <thead>
                            <tr>
                                <th style={styles.th}>Timestamp</th>
                                <th style={styles.th}>User</th>
                                <th style={styles.th}>Department</th>
                                <th style={styles.th}>Status</th>
                                <th style={styles.th}>Mode</th>
                                <th style={styles.th}>Templates</th>
                            </tr>
                        </thead>
                        <tbody>
                            {logs.slice(0, 10).map((log) => (
                                <tr key={log.log_id}>
                                    <td style={styles.td}>{formatDate(log.sent_at)}</td>
                                    <td style={styles.td}>{log.username}</td>
                                    <td style={styles.td}>{log.dept_code || 'UNMAPPED'}</td>
                                    <td style={styles.td}>{log.status}</td>
                                    <td style={styles.td}>{log.mode}</td>
                                    <td style={styles.td}>{log.template_ids.join(', ') || '-'}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                ) : (
                    <div style={styles.empty}>No logs are available.</div>
                )}
            </section>
        </div>
    );
}








