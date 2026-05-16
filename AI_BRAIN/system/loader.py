"""
AI_BRAIN Content Loader for GoAnsuran WhatsApp Chatbot.

Loads all txt and JSON files from the AI_BRAIN directory structure
into a dictionary organized by section.

Usage:
    from loader import load_ai_brain
    data = load_ai_brain("AI_BRAIN")
    print(data["core"]["identity"])  # content of core/identity.txt
"""

import os
import json
import glob


def _read_txt_file(filepath):
    """Read a text file and return its content as a string."""
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            return f.read().strip()
    except Exception as e:
        print(f"Warning: Could not read {filepath}: {e}")
        return ""


def _read_json_file(filepath):
    """Read a JSON file and return parsed content."""
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            return json.load(f)
    except Exception as e:
        print(f"Warning: Could not parse {filepath}: {e}")
        return {}


def _load_txt_directory(dirpath):
    """Load all .txt files from a directory into a dict keyed by filename (without extension)."""
    result = {}
    if not os.path.isdir(dirpath):
        print(f"Warning: Directory not found: {dirpath}")
        return result
    for filepath in glob.glob(os.path.join(dirpath, "*.txt")):
        name = os.path.splitext(os.path.basename(filepath))[0]
        result[name] = _read_txt_file(filepath)
    return result


def _load_product_directory(dirpath):
    """Load all .json files from brand subdirectories into a dict keyed by brand."""
    result = {}
    if not os.path.isdir(dirpath):
        print(f"Warning: Directory not found: {dirpath}")
        return result
    for brand_dir in sorted(os.listdir(dirpath)):
        brand_path = os.path.join(dirpath, brand_dir)
        if os.path.isdir(brand_path):
            result[brand_dir] = {}
            for json_file in sorted(glob.glob(os.path.join(brand_path, "*.json"))):
                model_name = os.path.splitext(os.path.basename(json_file))[0]
                result[brand_dir][model_name] = _read_json_file(json_file)
        elif brand_dir.endswith(".json"):
            # Handle flat JSON files (no brand subdirectories)
            name = os.path.splitext(os.path.basename(brand_path))[0]
            result[name] = _read_json_file(brand_path)
    return result


def load_ai_brain(base_path="AI_BRAIN"):
    """
    Load the entire AI_BRAIN directory structure into a dictionary.

    Args:
        base_path: Path to the AI_BRAIN directory (default: "AI_BRAIN")

    Returns:
        dict: Structured data with keys: core, flows, knowledge, templates, memory
    """
    data = {}

    # Load core files
    data["core"] = _load_txt_directory(os.path.join(base_path, "core"))

    # Load flow files
    data["flows"] = _load_txt_directory(os.path.join(base_path, "flows"))

    # Load knowledge files
    knowledge_path = os.path.join(base_path, "knowledge")
    data["knowledge"] = {
        "general": _load_txt_directory(os.path.join(knowledge_path, "general")),
        "goangkasa": {
            "overviews": _load_txt_directory(os.path.join(knowledge_path, "goangkasa")),
            "products": _load_product_directory(
                os.path.join(knowledge_path, "goangkasa", "products")
            ),
        },
        "goflexi": {
            "overviews": _load_txt_directory(os.path.join(knowledge_path, "goflexi")),
            "products": _load_product_directory(
                os.path.join(knowledge_path, "goflexi", "products")
            ),
        },
    }

    # Load template files
    data["templates"] = _load_txt_directory(os.path.join(base_path, "templates"))

    # Load memory templates
    memory_path = os.path.join(base_path, "memory")
    data["memory"] = {}
    if os.path.isdir(memory_path):
        for json_file in glob.glob(os.path.join(memory_path, "*.json")):
            name = os.path.splitext(os.path.basename(json_file))[0]
            data["memory"][name] = _read_json_file(json_file)

    # Load system master prompt
    system_path = os.path.join(base_path, "system")
    data["system"] = {}
    master_prompt = os.path.join(system_path, "master_prompt.txt")
    if os.path.isfile(master_prompt):
        data["system"]["master_prompt"] = _read_txt_file(master_prompt)

    return data


if __name__ == "__main__":
    import sys

    base = sys.argv[1] if len(sys.argv) > 1 else "AI_BRAIN"
    data = load_ai_brain(base)
    print(f"Loaded AI_BRAIN from: {base}")
    print(f"Core files: {len(data['core'])}")
    print(f"Flow files: {len(data['flows'])}")
    print(f"Knowledge sections: {list(data['knowledge'].keys())}")
    print(f"GoAngkasa brands: {len(data['knowledge']['goangkasa']['products'])}")
    print(f"GoFlexi brands: {len(data['knowledge']['goflexi']['products'])}")
    print(f"Templates: {len(data['templates'])}")
    print(f"Memory: {list(data['memory'].keys())}")
