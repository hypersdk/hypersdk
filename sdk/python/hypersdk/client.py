"""HyperSDK synchronous and asynchronous clients."""

import requests
from typing import List, Optional, Dict, Any
from urllib.parse import urljoin

from .models import (
    Job,
    JobDefinition,
    JobStatus,
    JobProgress,
    ScheduledJob,
    Webhook,
    DaemonStatus,
)
from .exceptions import (
    HyperSDKError,
    AuthenticationError,
    JobNotFoundError,
    APIError,
)


class HyperSDK:
    """Synchronous HyperSDK client.

    Example:
        >>> client = HyperSDK("http://localhost:8080")
        >>> client.login("admin", "password")
        >>> job_def = JobDefinition(
        ...     vm_path="/Datacenter/vm/my-vm",
        ...     output_dir="/exports"
        ... )
        >>> job_id = client.submit_job(job_def)
        >>> job = client.get_job(job_id)
    """

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        timeout: int = 30,
        verify_ssl: bool = True,
    ):
        """Initialize the HyperSDK client.

        Args:
            base_url: Base URL of the HyperSDK API (e.g., "http://localhost:8080")
            api_key: Optional API key for authentication
            timeout: Request timeout in seconds
            verify_ssl: Whether to verify SSL certificates
        """
        self.base_url = base_url.rstrip("/")
        self.api_key = api_key
        self.timeout = timeout
        self.verify_ssl = verify_ssl
        self.session = requests.Session()
        self._token: Optional[str] = None

        if api_key:
            self.session.headers["X-API-Key"] = api_key

    def _url(self, path: str) -> str:
        """Construct full URL from path."""
        return urljoin(self.base_url, path)

    def _request(
        self,
        method: str,
        path: str,
        json: Optional[Dict] = None,
        params: Optional[Dict] = None,
        **kwargs,
    ) -> Any:
        """Make HTTP request to the API.

        Args:
            method: HTTP method
            path: API path
            json: JSON body
            params: Query parameters
            **kwargs: Additional arguments for requests

        Returns:
            Response data

        Raises:
            APIError: If the request fails
        """
        url = self._url(path)
        headers = {}

        if self._token:
            headers["Authorization"] = f"Bearer {self._token}"

        try:
            response = self.session.request(
                method,
                url,
                json=json,
                params=params,
                headers=headers,
                timeout=self.timeout,
                verify=self.verify_ssl,
                **kwargs,
            )

            if response.status_code == 404:
                raise JobNotFoundError(f"Resource not found: {path}")

            if response.status_code == 401:
                raise AuthenticationError("Authentication failed")

            if not response.ok:
                error_msg = response.text
                try:
                    error_data = response.json()
                    error_msg = error_data.get("error", error_msg)
                except Exception:
                    pass
                raise APIError(
                    f"API error: {error_msg}",
                    status_code=response.status_code,
                    response=response.json() if response.text else None,
                )

            if response.text:
                return response.json()
            return None

        except requests.RequestException as e:
            raise APIError(f"Request failed: {str(e)}")

    # Authentication

    def login(self, username: str, password: str) -> str:
        """Login and obtain session token.

        Args:
            username: Username
            password: Password

        Returns:
            Session token

        Raises:
            AuthenticationError: If login fails
        """
        response = self._request(
            "POST",
            "/api/login",
            json={"username": username, "password": password},
        )
        self._token = response["token"]
        return self._token

    def logout(self) -> None:
        """Logout and invalidate session token."""
        self._request("POST", "/api/logout")
        self._token = None

    # Health & Status

    def health(self) -> Dict[str, Any]:
        """Check API health.

        Returns:
            Health status
        """
        return self._request("GET", "/health")

    def status(self) -> DaemonStatus:
        """Get daemon status.

        Returns:
            Daemon status information
        """
        data = self._request("GET", "/status")
        return DaemonStatus.from_dict(data)

    def capabilities(self) -> Dict[str, Any]:
        """Get export capabilities.

        Returns:
            Export capabilities
        """
        return self._request("GET", "/capabilities")

    # Job Management

    def submit_job(self, job_def: JobDefinition) -> str:
        """Submit a single job.

        Args:
            job_def: Job definition

        Returns:
            Job ID

        Raises:
            APIError: If submission fails
        """
        response = self._request("POST", "/jobs/submit", json=job_def.to_dict())
        if response["accepted"] == 0:
            errors = response.get("errors", ["Unknown error"])
            raise APIError(f"Job submission failed: {errors[0]}")
        return response["job_ids"][0]

    def submit_jobs(self, job_defs: List[JobDefinition]) -> List[str]:
        """Submit multiple jobs.

        Args:
            job_defs: List of job definitions

        Returns:
            List of job IDs

        Raises:
            APIError: If submission fails
        """
        jobs_data = [job.to_dict() for job in job_defs]
        response = self._request("POST", "/jobs/submit", json=jobs_data)
        return response["job_ids"]

    def get_job(self, job_id: str) -> Job:
        """Get job details.

        Args:
            job_id: Job ID

        Returns:
            Job information

        Raises:
            JobNotFoundError: If job not found
        """
        data = self._request("GET", f"/jobs/{job_id}")
        return Job.from_dict(data)

    def query_jobs(
        self,
        job_ids: Optional[List[str]] = None,
        status: Optional[List[JobStatus]] = None,
        all: bool = False,
        limit: Optional[int] = None,
    ) -> List[Job]:
        """Query jobs with filters.

        Args:
            job_ids: Filter by specific job IDs
            status: Filter by job status
            all: Return all jobs
            limit: Limit number of results

        Returns:
            List of jobs
        """
        request_data = {}
        if job_ids:
            request_data["job_ids"] = job_ids
        if status:
            request_data["status"] = [s.value for s in status]
        if all:
            request_data["all"] = True
        if limit:
            request_data["limit"] = limit

        response = self._request("POST", "/jobs/query", json=request_data)
        return [Job.from_dict(job_data) for job_data in response["jobs"]]

    def list_jobs(self, all: bool = True) -> List[Job]:
        """List all jobs.

        Args:
            all: Return all jobs

        Returns:
            List of all jobs
        """
        params = {"all": "true"} if all else {}
        response = self._request("GET", "/jobs/query", params=params)
        return [Job.from_dict(job_data) for job_data in response["jobs"]]

    def cancel_job(self, job_id: str) -> bool:
        """Cancel a running job.

        Args:
            job_id: Job ID to cancel

        Returns:
            True if cancelled successfully

        Raises:
            APIError: If cancellation fails
        """
        response = self._request("POST", "/jobs/cancel", json={"job_ids": [job_id]})
        if job_id in response["cancelled"]:
            return True
        if job_id in response["failed"]:
            error = response["errors"].get(job_id, "Unknown error")
            raise APIError(f"Failed to cancel job: {error}")
        return False

    def cancel_jobs(self, job_ids: List[str]) -> Dict[str, Any]:
        """Cancel multiple jobs.

        Args:
            job_ids: List of job IDs to cancel

        Returns:
            Cancel results with cancelled and failed lists
        """
        return self._request("POST", "/jobs/cancel", json={"job_ids": job_ids})

    def get_job_progress(self, job_id: str) -> JobProgress:
        """Get job progress.

        Args:
            job_id: Job ID

        Returns:
            Job progress information
        """
        data = self._request("GET", f"/jobs/progress/{job_id}")
        return JobProgress.from_dict(data)

    def get_job_logs(self, job_id: str) -> str:
        """Get job logs.

        Args:
            job_id: Job ID

        Returns:
            Job logs as string
        """
        url = self._url(f"/jobs/logs/{job_id}")
        response = self.session.get(
            url,
            headers={"Authorization": f"Bearer {self._token}"} if self._token else {},
            timeout=self.timeout,
            verify=self.verify_ssl,
        )
        if not response.ok:
            raise APIError(f"Failed to get logs: {response.text}")
        return response.text

    def get_job_eta(self, job_id: str) -> str:
        """Get job estimated time of arrival.

        Args:
            job_id: Job ID

        Returns:
            ETA string
        """
        data = self._request("GET", f"/jobs/eta/{job_id}")
        return data.get("estimated_remaining", "Unknown")

    # VM Operations

    def list_vms(self, vcenter_config: Dict[str, Any]) -> List[Dict[str, Any]]:
        """List VMs from vCenter.

        Args:
            vcenter_config: vCenter configuration

        Returns:
            List of VMs
        """
        return self._request("POST", "/vms/list", json=vcenter_config)

    def get_vm_info(self, vcenter_config: Dict[str, Any], vm_path: str) -> Dict[str, Any]:
        """Get VM information.

        Args:
            vcenter_config: vCenter configuration
            vm_path: VM path

        Returns:
            VM information
        """
        return self._request(
            "POST",
            "/vms/info",
            json={"vcenter": vcenter_config, "vm_path": vm_path},
        )

    def shutdown_vm(self, vcenter_config: Dict[str, Any], vm_path: str) -> Dict[str, Any]:
        """Shutdown a VM.

        Args:
            vcenter_config: vCenter configuration
            vm_path: VM path

        Returns:
            Shutdown status
        """
        return self._request(
            "POST",
            "/vms/shutdown",
            json={"vcenter": vcenter_config, "vm_path": vm_path},
        )

    # Schedule Management

    def list_schedules(self) -> List[ScheduledJob]:
        """List all scheduled jobs.

        Returns:
            List of scheduled jobs
        """
        data = self._request("GET", "/schedules")
        return [ScheduledJob.from_dict(s) for s in data]

    def create_schedule(self, schedule: ScheduledJob) -> ScheduledJob:
        """Create a new scheduled job.

        Args:
            schedule: Scheduled job configuration

        Returns:
            Created scheduled job
        """
        data = self._request("POST", "/schedules", json=schedule.to_dict())
        return ScheduledJob.from_dict(data)

    def get_schedule(self, schedule_id: str) -> ScheduledJob:
        """Get schedule details.

        Args:
            schedule_id: Schedule ID

        Returns:
            Scheduled job
        """
        data = self._request("GET", f"/schedules/{schedule_id}")
        return ScheduledJob.from_dict(data)

    def update_schedule(self, schedule_id: str, schedule: ScheduledJob) -> ScheduledJob:
        """Update a schedule.

        Args:
            schedule_id: Schedule ID
            schedule: Updated schedule configuration

        Returns:
            Updated scheduled job
        """
        data = self._request("PUT", f"/schedules/{schedule_id}", json=schedule.to_dict())
        return ScheduledJob.from_dict(data)

    def delete_schedule(self, schedule_id: str) -> None:
        """Delete a schedule.

        Args:
            schedule_id: Schedule ID
        """
        self._request("DELETE", f"/schedules/{schedule_id}")

    def enable_schedule(self, schedule_id: str) -> None:
        """Enable a schedule.

        Args:
            schedule_id: Schedule ID
        """
        self._request("POST", f"/schedules/{schedule_id}/enable")

    def disable_schedule(self, schedule_id: str) -> None:
        """Disable a schedule.

        Args:
            schedule_id: Schedule ID
        """
        self._request("POST", f"/schedules/{schedule_id}/disable")

    def trigger_schedule(self, schedule_id: str) -> None:
        """Manually trigger a schedule.

        Args:
            schedule_id: Schedule ID
        """
        self._request("POST", f"/schedules/{schedule_id}/trigger")

    # Webhook Management

    def list_webhooks(self) -> List[Webhook]:
        """List all webhooks.

        Returns:
            List of webhooks
        """
        data = self._request("GET", "/webhooks")
        return [Webhook.from_dict(w) for w in data]

    def add_webhook(self, webhook: Webhook) -> None:
        """Add a new webhook.

        Args:
            webhook: Webhook configuration
        """
        self._request("POST", "/webhooks", json=webhook.to_dict())

    def test_webhook(self, url: str) -> None:
        """Test a webhook URL.

        Args:
            url: Webhook URL to test

        Raises:
            APIError: If webhook test fails
        """
        self._request("POST", "/webhooks/test", json={"url": url})

    def delete_webhook(self, webhook_id: str) -> None:
        """Delete a webhook.

        Args:
            webhook_id: Webhook ID
        """
        self._request("DELETE", f"/webhooks/{webhook_id}")

    # Libvirt Operations

    def list_domains(self) -> List[Dict[str, Any]]:
        """List libvirt domains.

        Returns:
            List of domains
        """
        return self._request("GET", "/libvirt/domains")

    def get_domain(self, name: str) -> Dict[str, Any]:
        """Get domain details.

        Args:
            name: Domain name

        Returns:
            Domain information
        """
        return self._request("GET", "/libvirt/domain", params={"name": name})

    def start_domain(self, name: str) -> None:
        """Start a domain.

        Args:
            name: Domain name
        """
        self._request("POST", "/libvirt/domain/start", json={"name": name})

    def shutdown_domain(self, name: str) -> None:
        """Shutdown a domain.

        Args:
            name: Domain name
        """
        self._request("POST", "/libvirt/domain/shutdown", json={"name": name})

    def list_snapshots(self, domain: str) -> List[Dict[str, Any]]:
        """List snapshots for a domain.

        Args:
            domain: Domain name

        Returns:
            List of snapshots
        """
        return self._request("GET", "/libvirt/snapshots", params={"domain": domain})

    def create_snapshot(
        self, domain: str, name: str, description: Optional[str] = None
    ) -> None:
        """Create a snapshot.

        Args:
            domain: Domain name
            name: Snapshot name
            description: Optional snapshot description
        """
        data = {"domain": domain, "name": name}
        if description:
            data["description"] = description
        self._request("POST", "/libvirt/snapshot/create", json=data)

    # Incremental Export & Changed Block Tracking

    def enable_cbt(self, vm_path: str) -> Dict[str, Any]:
        """Enable Changed Block Tracking (CBT) on a VM.

        Args:
            vm_path: VM path

        Returns:
            Response with success status and message
        """
        return self._request("POST", "/cbt/enable", json={"vm_path": vm_path})

    def disable_cbt(self, vm_path: str) -> Dict[str, Any]:
        """Disable Changed Block Tracking (CBT) on a VM.

        Args:
            vm_path: VM path

        Returns:
            Response with success status and message
        """
        return self._request("POST", "/cbt/disable", json={"vm_path": vm_path})

    def get_cbt_status(self, vm_path: str) -> Dict[str, Any]:
        """Get CBT status and incremental export information for a VM.

        Args:
            vm_path: VM path

        Returns:
            CBT status including:
                - cbt_enabled: Whether CBT is enabled
                - disks: List of disk metadata
                - last_export: Last export metadata (if any)
                - can_incremental: Whether incremental export is possible
                - reason: Reason why incremental is/isn't possible
        """
        return self._request("POST", "/cbt/status", json={"vm_path": vm_path})

    def analyze_incremental_export(self, vm_path: str) -> Dict[str, Any]:
        """Analyze incremental export potential for a VM.

        Args:
            vm_path: VM path

        Returns:
            Analysis including:
                - can_incremental: Whether incremental export is possible
                - reason: Reason why incremental is/isn't possible
                - last_export: Last export metadata (if any)
                - current_disks: Current disk metadata
                - estimated_savings_bytes: Estimated savings from incremental export
                - estimated_duration: Estimated duration
        """
        return self._request("POST", "/incremental/analyze", json={"vm_path": vm_path})

    # Hyper2KVM Integration

    def convert_vm(self, source_path: str, output_path: str) -> str:
        """Convert VM using hyper2kvm.

        Args:
            source_path: Source VM path
            output_path: Output path for converted VM

        Returns:
            Conversion ID
        """
        response = self._request(
            "POST",
            "/convert/vm",
            json={"source_path": source_path, "output_path": output_path},
        )
        return response["conversion_id"]

    def get_conversion_status(self, conversion_id: str) -> Dict[str, Any]:
        """Get VM conversion status.

        Args:
            conversion_id: Conversion ID

        Returns:
            Conversion status
        """
        return self._request(
            "GET", "/convert/status", params={"conversion_id": conversion_id}
        )

    def close(self) -> None:
        """Close the client session."""
        self.session.close()

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()


# Alias for backward compatibility
AsyncHyperSDK = HyperSDK  # TODO: Implement true async client with aiohttp
