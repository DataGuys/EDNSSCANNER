// main.js - JavaScript functionality for DNS Subdomain Scanner

document.addEventListener('DOMContentLoaded', function() {
    // Add DataTables if available
    if (typeof(jQuery) !== 'undefined' && typeof(jQuery.fn.DataTable) !== 'undefined') {
        jQuery('table.table').DataTable({
            "pageLength": 10,
            "lengthMenu": [[10, 25, 50, 100, -1], [10, 25, 50, 100, "All"]],
            "order": [[0, "asc"]]
        });
    }

    // Form validation
    const scanForm = document.querySelector('form[action="/scan"]');
    if (scanForm) {
        scanForm.addEventListener('submit', function(event) {
            const domainInput = document.getElementById('domain');
            const domainValue = domainInput.value.trim();
            
            // Basic domain validation (more complex validation happens server-side)
            if (!isDomainValid(domainValue)) {
                event.preventDefault();
                alert('Please enter a valid domain name (e.g., example.com)');
                domainInput.focus();
            }
        });
    }
    
    // Domain validation helper function
    function isDomainValid(domain) {
        // Simple regex to check for valid domain format
        // Allows subdomains and TLDs of various lengths
        const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/;
        
        // Also accept domain without protocol or www prefix
        const cleanDomain = domain
            .replace(/^https?:\/\//, '')
            .replace(/^www\./, '');
            
        return domainRegex.test(cleanDomain);
    }
    
    // Handle auto-refresh for running scans
    const jobStatus = document.querySelector('.card-body .badge');
    if (jobStatus && jobStatus.textContent.trim() === 'Running') {
        setTimeout(function() {
            window.location.reload();
        }, 5000); // Refresh every 5 seconds
    }
    
    // Add confirmation for CSV download on large result sets
    const csvLinks = document.querySelectorAll('a[href$="/csv"]');
    csvLinks.forEach(function(link) {
        link.addEventListener('click', function(event) {
            const resultsCount = document.querySelectorAll('#resultsTable tbody tr').length;
            if (resultsCount > 1000) {
                if (!confirm(`You're about to download a CSV with ${resultsCount} results. This may take a moment. Continue?`)) {
                    event.preventDefault();
                }
            }
        });
    });
    
    // Add tooltips for record types
    const recordTypeBadges = document.querySelectorAll('.badge');
    recordTypeBadges.forEach(function(badge) {
        let tooltip = '';
        switch (badge.textContent) {
            case 'A':
                tooltip = 'IPv4 Address';
                break;
            case 'AAAA':
                tooltip = 'IPv6 Address';
                break;
            case 'CNAME':
                tooltip = 'Canonical Name (Alias)';
                break;
            case 'MX':
                tooltip = 'Mail Exchange';
                break;
            case 'TXT':
                tooltip = 'Text Record';
                break;
            case 'NS':
                tooltip = 'Name Server';
                break;
            case 'SOA':
                tooltip = 'Start of Authority';
                break;
        }
        
        if (tooltip) {
            badge.title = tooltip;
            // Add Bootstrap tooltip if available
            if (typeof(bootstrap) !== 'undefined' && typeof(bootstrap.Tooltip) !== 'undefined') {
                new bootstrap.Tooltip(badge);
            }
        }
    });
});