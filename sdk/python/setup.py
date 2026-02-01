"""Setup script for hypersdk Python package."""

from setuptools import setup, find_packages

with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="hypersdk",
    version="1.0.0",
    author="HyperSDK Contributors",
    author_email="",
    description="Python client library for HyperSDK VM migration platform",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/ssahani/hypersdk",
    packages=find_packages(),
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Intended Audience :: System Administrators",
        "License :: OSI Approved :: GNU Lesser General Public License v3 or later (LGPLv3+)",
        "Operating System :: OS Independent",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
        "Topic :: System :: Systems Administration",
        "Topic :: Software Development :: Libraries :: Python Modules",
    ],
    python_requires=">=3.8",
    install_requires=[
        "requests>=2.25.0",
    ],
    extras_require={
        "dev": [
            "pytest>=7.0.0",
            "pytest-cov>=3.0.0",
            "black>=22.0.0",
            "mypy>=0.990",
            "types-requests",
        ],
    },
    keywords="vm migration export vmware vsphere aws azure gcp hypervisor kvm libvirt",
    project_urls={
        "Bug Reports": "https://github.com/ssahani/hypersdk/issues",
        "Source": "https://github.com/ssahani/hypersdk",
        "Documentation": "https://github.com/ssahani/hypersdk/blob/main/sdk/python/README.md",
    },
)
