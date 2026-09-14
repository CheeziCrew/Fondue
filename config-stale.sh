#!/bin/bash
# config-stale.sh — Detect stale version references in config repo
# Compares base-url versions in config.yaml files against appVersion in values.yaml
#
# Usage: bash config-stale.sh [config-repo-root] [output-file]
#   config-repo-root: path to config repo (default: current directory)
#   output-file:      path to result file (default: result.md)

set -euo pipefail

REPO_ROOT="${1:-.}"
OUTPUT="${2:-result.md}"
REPO_ROOT=$(cd "$REPO_ROOT" && pwd)

# Associative arrays (bash 4+)
declare -A ACTUAL_VERSIONS
declare -A NORMALIZED_LOOKUP  # normalized-name → canonical service name
declare -a ALL_ENVS

# Raw stale data lines: "env|source|target|config_ver|actual_ver"
declare -a STALE_RECORDS
# Raw unresolved lines: "env|source|url_target|config_ver"
declare -a UNRESOLVED_RECORDS

total_refs=0
stale_count=0
up_to_date_count=0
unresolved_count=0
services_scanned=0

# Normalize a service name: lowercase, strip hyphens/underscores
normalize() {
    echo "$1" | tr '[:upper:]' '[:lower:]' | tr -d '-' | tr -d '_'
}

# --- Step 1: Collect actual versions from */deployment/values.yaml ---

collect_versions() {
    local values_file service version norm
    while IFS= read -r values_file; do
        service=$(echo "$values_file" | cut -d'/' -f2)
        version=$(grep -m1 'appVersion:' "$REPO_ROOT/${values_file#./}" 2>/dev/null | sed 's/.*appVersion:[[:space:]]*//' | tr -d '"' | tr -d "'" | tr -d '[:space:]')
        if [[ -n "$version" ]]; then
            ACTUAL_VERSIONS["$service"]="$version"
            norm=$(normalize "$service")
            NORMALIZED_LOOKUP["$norm"]="$service"
            ((services_scanned++)) || true
        fi
    done < <(cd "$REPO_ROOT" && find . -maxdepth 3 -name "values.yaml" -path "*/deployment/*" -type f 2>/dev/null)
}

# Resolve a target name to a known service (exact first, then fuzzy)
resolve_service() {
    local target="$1"
    if [[ -n "${ACTUAL_VERSIONS[$target]:-}" ]]; then
        echo "$target"
        return
    fi
    local norm
    norm=$(normalize "$target")
    if [[ -n "${NORMALIZED_LOOKUP[$norm]:-}" ]]; then
        echo "${NORMALIZED_LOOKUP[$norm]}"
        return
    fi
    echo ""
}

# --- Step 2: Scan config references and compare ---

scan_and_compare() {
    local config_file source_service env_name
    while IFS= read -r config_file; do
        source_service=$(echo "$config_file" | cut -d'/' -f2)
        env_name=$(echo "$config_file" | cut -d'/' -f5)

        if [[ ! " ${ALL_ENVS[*]:-} " =~ " $env_name " ]]; then
            ALL_ENVS+=("$env_name")
        fi

        while IFS= read -r line; do
            local url
            url=$(echo "$line" | sed -n 's/.*\(https\{0,1\}:\/\/[^[:space:]"'"'"']*\).*/\1/p')
            [[ -z "$url" ]] && continue
            url="${url%/}"

            local target_service config_version
            config_version=$(echo "$url" | awk -F'/' '{print $NF}')
            target_service=$(echo "$url" | awk -F'/' '{print $(NF-1)}')

            [[ ! "$config_version" =~ [0-9] ]] && continue
            [[ -z "$target_service" ]] && continue
            [[ "$target_service" =~ \. ]] && continue

            ((total_refs++)) || true

            local resolved
            resolved=$(resolve_service "$target_service")

            if [[ -z "$resolved" ]]; then
                ((unresolved_count++)) || true
                UNRESOLVED_RECORDS+=("${env_name}|${source_service}|${target_service}|${config_version}")
                continue
            fi

            local actual="${ACTUAL_VERSIONS[$resolved]}"
            if [[ "$config_version" != "$actual" ]]; then
                ((stale_count++)) || true
                STALE_RECORDS+=("${env_name}|${source_service}|${resolved}|${config_version}|${actual}")
            else
                ((up_to_date_count++)) || true
            fi
        done < <(grep -i 'url:' "$REPO_ROOT/${config_file#./}" 2>/dev/null | grep 'https\{0,1\}://' || true)

    done < <(cd "$REPO_ROOT" && find . -name "config.yaml" -path "*/deployment/config/*" -type f 2>/dev/null)
}

# --- Step 3: Generate report ---

generate_report() {
    local date_str
    date_str=$(date +%Y-%m-%d)
    local matched_refs=$((up_to_date_count + stale_count))
    local stale_pct=0
    if [[ $matched_refs -gt 0 ]]; then
        stale_pct=$((stale_count * 100 / matched_refs))
    fi

    cat > "$OUTPUT" <<EOF
# Config Version Staleness Report
> Generated: $date_str | Scanned: $REPO_ROOT

## Summary

| Metric | Count |
|---|---|
| Services with versions | $services_scanned |
| URL references checked | $total_refs |
| Up to date | $up_to_date_count |
| **Stale** | **$stale_count** ($stale_pct%) |
| Unresolved (no matching service) | $unresolved_count |
EOF

    if [[ $stale_count -eq 0 ]]; then
        echo "" >> "$OUTPUT"
        echo "All config references are up to date." >> "$OUTPUT"
        return
    fi

    # --- Quick Wins: which target updates fix the most refs? ---
    # Count stale refs per target service (across all envs, deduplicated by source)
    # Use production env only for quick-wins to avoid double-counting
    local quick_win_env="${ALL_ENVS[0]}"
    declare -A TARGET_COUNTS
    declare -A TARGET_ACTUAL
    for record in "${STALE_RECORDS[@]}"; do
        local r_env r_source r_target r_cfgver r_actver
        IFS='|' read -r r_env r_source r_target r_cfgver r_actver <<< "$record"
        # Count each target across one env to avoid double-counting
        if [[ "$r_env" == "$quick_win_env" ]]; then
            TARGET_COUNTS["$r_target"]=$(( ${TARGET_COUNTS[$r_target]:-0} + 1 ))
            TARGET_ACTUAL["$r_target"]="$r_actver"
        fi
    done

    # Sort targets by count (descending)
    echo "" >> "$OUTPUT"
    echo "## Quick Wins" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "Updating these services would resolve the most stale references:" >> "$OUTPUT"
    echo "" >> "$OUTPUT"
    echo "| Target Service | Current Version | Stale Refs | Impact |" >> "$OUTPUT"
    echo "|---|---|---|---|" >> "$OUTPUT"

    # Sort by count descending using a temp approach
    local sort_data=""
    for target in "${!TARGET_COUNTS[@]}"; do
        local cnt="${TARGET_COUNTS[$target]}"
        sort_data+="${cnt}|${target}|${TARGET_ACTUAL[$target]}"$'\n'
    done
    # Sort numerically descending, take top entries
    echo -n "$sort_data" | sort -t'|' -k1 -rn | while IFS='|' read -r cnt target actver; do
        [[ -z "$cnt" ]] && continue
        local impact="low"
        if [[ $cnt -ge 10 ]]; then
            impact="high"
        elif [[ $cnt -ge 5 ]]; then
            impact="medium"
        fi
        echo "| $target | $actver | $cnt services behind | $impact |" >> "$OUTPUT"
    done

    # --- Stale details grouped by target, sorted by severity ---
    echo "" >> "$OUTPUT"
    echo "## Stale Details" >> "$OUTPUT"

    local sorted_envs
    IFS=$'\n' sorted_envs=($(printf '%s\n' "${ALL_ENVS[@]}" | sort)); unset IFS

    for env in "${sorted_envs[@]}"; do
        # Collect records for this env
        local env_records=""
        for record in "${STALE_RECORDS[@]}"; do
            local r_env r_source r_target r_cfgver r_actver
            IFS='|' read -r r_env r_source r_target r_cfgver r_actver <<< "$record"
            [[ "$r_env" != "$env" ]] && continue
            env_records+="$record"$'\n'
        done
        [[ -z "$env_records" ]] && continue

        echo "" >> "$OUTPUT"
        echo "### $env" >> "$OUTPUT"

        # Group by target service, sorted by number of affected sources (desc)
        declare -A ENV_TARGET_SOURCES
        declare -A ENV_TARGET_LINES
        while IFS='|' read -r r_env r_source r_target r_cfgver r_actver; do
            [[ -z "$r_target" ]] && continue
            ENV_TARGET_SOURCES["$r_target"]=$(( ${ENV_TARGET_SOURCES[$r_target]:-0} + 1 ))
            ENV_TARGET_LINES["$r_target"]+="| $r_source | $r_cfgver | $r_actver |"$'\n'
        done <<< "$env_records"

        # Sort targets by count descending
        local target_sort=""
        for target in "${!ENV_TARGET_SOURCES[@]}"; do
            target_sort+="${ENV_TARGET_SOURCES[$target]}|${target}"$'\n'
        done

        echo -n "$target_sort" | sort -t'|' -k1 -rn | while IFS='|' read -r cnt target; do
            [[ -z "$target" ]] && continue
            local actual="${ACTUAL_VERSIONS[$target]}"
            echo "" >> "$OUTPUT"
            echo "**$target** (current: $actual) — $cnt service(s) behind" >> "$OUTPUT"
            echo "" >> "$OUTPUT"
            echo "| Source Service | Config Version | Actual Version |" >> "$OUTPUT"
            echo "|---|---|---|" >> "$OUTPUT"
            echo -n "${ENV_TARGET_LINES[$target]}" >> "$OUTPUT"
        done

        # Clean up for next env
        unset ENV_TARGET_SOURCES
        unset ENV_TARGET_LINES
        declare -A ENV_TARGET_SOURCES
        declare -A ENV_TARGET_LINES
    done

    # --- Unresolved ---
    if [[ $unresolved_count -gt 0 ]]; then
        echo "" >> "$OUTPUT"
        echo "## Unresolved References" >> "$OUTPUT"
        echo "" >> "$OUTPUT"
        echo "These URL targets could not be matched to any known service:" >> "$OUTPUT"

        for env in "${sorted_envs[@]}"; do
            local has_entries=false
            for record in "${UNRESOLVED_RECORDS[@]}"; do
                local r_env r_source r_target r_cfgver
                IFS='|' read -r r_env r_source r_target r_cfgver <<< "$record"
                if [[ "$r_env" == "$env" ]]; then
                    if [[ "$has_entries" == false ]]; then
                        echo "" >> "$OUTPUT"
                        echo "### $env" >> "$OUTPUT"
                        echo "" >> "$OUTPUT"
                        echo "| Source Service | URL Target | Config Version |" >> "$OUTPUT"
                        echo "|---|---|---|" >> "$OUTPUT"
                        has_entries=true
                    fi
                    echo "| $r_source | $r_target | $r_cfgver |" >> "$OUTPUT"
                fi
            done
        done
    fi

    echo "" >> "$OUTPUT"
    echo "---" >> "$OUTPUT"
    echo "*Generated by config-stale.sh*" >> "$OUTPUT"
}

# --- Main ---

echo "Scanning $REPO_ROOT ..."

collect_versions
echo "  Found ${services_scanned} services with versions"

scan_and_compare
echo "  Checked ${total_refs} URL references"
echo "  Found ${stale_count} stale references"
echo "  Found ${unresolved_count} unresolved references"

generate_report
echo "Report written to $OUTPUT"
