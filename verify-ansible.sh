#!/bin/bash
cd kibaship-ansible
source venv/bin/activate

echo "=== ENVIRONMENT CHECK ==="
python --version
ansible --version | head -n1
molecule --version

echo -e "\n=== ROLE STRUCTURE ==="
echo "Discovered roles:"
ls -1 roles/
ROLE_COUNT=$(ls -1 roles/ | wc -l)
echo "Total roles: $ROLE_COUNT"

# Get list of all roles with molecule scenarios
TESTABLE_ROLES=()
for role in roles/*/; do
    role_name=$(basename "$role")
    if [[ -d "$role/molecule/default" ]]; then
        TESTABLE_ROLES+=("$role_name")
    fi
done

echo -e "\n=== ROLES WITH MOLECULE TESTS ==="
printf '%s\n' "${TESTABLE_ROLES[@]}"
echo "Testable roles: ${#TESTABLE_ROLES[@]}"

echo -e "\n=== QUICK SYNTAX CHECK (ALL ROLES) ==="
SYNTAX_FAILED=()
for role in "${TESTABLE_ROLES[@]}"; do
    echo "--- Syntax check: $role ---"
    cd "roles/$role"
    if ! molecule syntax; then
        SYNTAX_FAILED+=("$role")
    fi
    cd - > /dev/null
done

if [[ ${#SYNTAX_FAILED[@]} -gt 0 ]]; then
    echo -e "\n❌ SYNTAX FAILURES:"
    printf '%s\n' "${SYNTAX_FAILED[@]}"
    exit 1
fi

echo -e "\n=== FULL TEST SUITE (ALL ROLES) ==="
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=()

for role in "${TESTABLE_ROLES[@]}"; do
    echo -e "\n=========================================="
    echo "🧪 TESTING ROLE: $role"
    echo "=========================================="
    cd "roles/$role"
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    if molecule test; then
        echo "✅ PASSED: $role"
        PASSED_TESTS=$((PASSED_TESTS + 1))
    else
        echo "❌ FAILED: $role"
        FAILED_TESTS+=("$role")
    fi
    cd - > /dev/null
done

echo -e "\n=========================================="
echo "🏆 FINAL TEST RESULTS"
echo "=========================================="
echo "Total roles tested: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: ${#FAILED_TESTS[@]}"

if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
    echo -e "\n❌ FAILED ROLES:"
    printf '%s\n' "${FAILED_TESTS[@]}"
    exit 1
else
    echo -e "\n🎉 ALL TESTS PASSED!"
    exit 0
fi