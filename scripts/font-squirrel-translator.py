#!/usr/bin/env python3
"""
Font Squirrel API Translator for FontGet

Fetches font data from Font Squirrel API and transforms it to FontGet format.
API: https://www.fontsquirrel.com/api/fontlist/all
"""

import json
import os
import requests
import time
from datetime import datetime
from typing import Dict, List, Any, Optional


class FontSquirrelTranslator:
    def __init__(self):
        """Initialize translator."""
        self.base_url = "https://www.fontsquirrel.com/api/fontlist/all"
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'FontGet/1.0 (https://github.com/graphixa/fontget)'
        })
    
    def fetch_fonts(self) -> List[Dict[str, Any]]:
        """Fetch all fonts from Font Squirrel API."""
        print("Fetching fonts from Font Squirrel API...")
        
        try:
            response = self.session.get(self.base_url, timeout=30)
            response.raise_for_status()
            
            # Font Squirrel returns a list of fonts directly
            fonts = response.json()
            print(f"Found {len(fonts)} fonts from Font Squirrel API")
            return fonts
            
        except requests.exceptions.RequestException as e:
            print(f"Error fetching fonts: {e}")
            return []
    
    def transform_font(self, font_data: Dict[str, Any]) -> Optional[Dict[str, Any]]:
        """Transform Font Squirrel data to FontGet format."""
        try:
            # Extract basic info
            family_name = font_data.get("family_name", "")
            if not family_name:
                return None
            
            # Create font ID
            font_id = f"squirrel.{family_name.lower().replace(' ', '-').replace('_', '-')}"
            
            # Extract license info
            license_info = font_data.get("license", {})
            license_type = license_info.get("name", "Unknown")
            license_url = license_info.get("url", "")
            
            # Extract designer info
            designer = font_data.get("designer", "")
            foundry = font_data.get("foundry", "")
            
            # Extract classification
            classification = font_data.get("classification", {})
            category = classification.get("name", "Other")
            
            # Extract tags
            tags = []
            if classification.get("name"):
                tags.append(classification["name"].lower().replace(" ", "-"))
            
            # Check if font is free
            is_free = font_data.get("is_free", False)
            if is_free:
                tags.append("free")
            else:
                tags.append("commercial")
            
            # Extract description
            description = font_data.get("description", "")
            if not description and font_data.get("short_description"):
                description = font_data["short_description"]
            
            # Extract version
            version = font_data.get("version", "1.0")
            
            # Extract last modified
            last_modified = font_data.get("updated_at", "")
            if last_modified:
                try:
                    # Parse ISO format and convert to our format
                    dt = datetime.fromisoformat(last_modified.replace('Z', '+00:00'))
                    last_modified = dt.isoformat() + "Z"
                except:
                    last_modified = ""
            
            # Generate source URL
            family_urlname = font_data.get("family_urlname", family_name.lower().replace(" ", "-"))
            source_url = f"https://www.fontsquirrel.com/fonts/{family_urlname}"
            
            # Extract variants (Font Squirrel has different structure)
            variants = self._extract_variants(font_data)
            
            # Calculate popularity (Font Squirrel doesn't provide this, so we'll use a simple heuristic)
            popularity = self._calculate_popularity(font_data)
            
            return {
                "name": family_name,
                "family": family_name,
                "license": license_type,
                "license_url": license_url,
                "designer": designer,
                "foundry": foundry,
                "version": version,
                "description": description,
                "categories": [category] if category != "Other" else [],
                "tags": tags,
                "popularity": popularity,
                "last_modified": last_modified,
                "metadata_url": f"https://www.fontsquirrel.com/api/familyinfo/{family_urlname}",
                "source_url": source_url,
                "variants": variants,
                "unicode_ranges": self._extract_unicode_ranges(font_data),
                "languages": self._extract_languages(font_data),
                "sample_text": "The quick brown fox jumps over the lazy dog"
            }
            
        except Exception as e:
            print(f"Warning: Failed to transform font {font_data.get('family_name', 'unknown')}: {e}")
            return None
    
    def _extract_variants(self, font_data: Dict[str, Any]) -> List[Dict[str, Any]]:
        """Extract font variants from Font Squirrel data."""
        variants = []
        
        # Font Squirrel has a different structure for variants
        # We'll create basic variants based on available information
        
        # Check if there are specific font files listed
        font_files = font_data.get("font_files", [])
        
        if font_files:
            # Process actual font files
            for file_info in font_files:
                variant_name = file_info.get("name", "Regular")
                weight = self._parse_weight(variant_name)
                style = self._parse_style(variant_name)
                
                # Generate file URLs
                files = {}
                if file_info.get("ttf_url"):
                    files["ttf"] = file_info["ttf_url"]
                if file_info.get("otf_url"):
                    files["otf"] = file_info["otf_url"]
                
                if files:  # Only add if we have actual files
                    variants.append({
                        "name": variant_name,
                        "weight": weight,
                        "style": style,
                        "subsets": ["latin"],  # Default subset
                        "files": files
                    })
        else:
            # Fallback: create basic variants based on font name
            family_name = font_data.get("family_name", "")
            
            # Common variant patterns
            variant_patterns = [
                ("Regular", 400, "normal"),
                ("Bold", 700, "normal"),
                ("Italic", 400, "italic"),
                ("Bold Italic", 700, "italic"),
                ("Light", 300, "normal"),
                ("Medium", 500, "normal"),
                ("Semi Bold", 600, "normal"),
                ("Extra Bold", 800, "normal"),
                ("Black", 900, "normal")
            ]
            
            for variant_name, weight, style in variant_patterns:
                # Generate file URLs (Font Squirrel pattern)
                family_urlname = font_data.get("family_urlname", family_name.lower().replace(" ", "-"))
                variant_urlname = variant_name.lower().replace(" ", "-")
                
                files = {}
                # Try common file patterns
                ttf_url = f"https://www.fontsquirrel.com/fonts/download/{family_urlname}/{variant_urlname}"
                files["ttf"] = ttf_url
                
                variants.append({
                    "name": f"{family_name} {variant_name}",
                    "weight": weight,
                    "style": style,
                    "subsets": ["latin"],
                    "files": files
                })
        
        return variants[:5]  # Limit to 5 variants to avoid too many
    
    def _parse_weight(self, variant_name: str) -> int:
        """Parse weight from variant name."""
        variant_lower = variant_name.lower()
        
        if "thin" in variant_lower or "hairline" in variant_lower:
            return 100
        elif "extra-light" in variant_lower or "ultra-light" in variant_lower:
            return 200
        elif "light" in variant_lower:
            return 300
        elif "regular" in variant_lower or "normal" in variant_lower:
            return 400
        elif "medium" in variant_lower:
            return 500
        elif "semi-bold" in variant_lower or "demi-bold" in variant_lower:
            return 600
        elif "bold" in variant_lower:
            return 700
        elif "extra-bold" in variant_lower or "ultra-bold" in variant_lower:
            return 800
        elif "black" in variant_lower or "heavy" in variant_lower:
            return 900
        else:
            return 400  # Default to regular
    
    def _parse_style(self, variant_name: str) -> str:
        """Parse style from variant name."""
        variant_lower = variant_name.lower()
        
        if "italic" in variant_lower or "oblique" in variant_lower:
            return "italic"
        else:
            return "normal"
    
    def _calculate_popularity(self, font_data: Dict[str, Any]) -> int:
        """Calculate popularity score (0-100) based on available data."""
        score = 50  # Base score
        
        # Bonus for free fonts (more likely to be popular)
        if font_data.get("is_free", False):
            score += 20
        
        # Bonus for having description
        if font_data.get("description") or font_data.get("short_description"):
            score += 10
        
        # Bonus for having designer info
        if font_data.get("designer"):
            score += 10
        
        # Bonus for having foundry info
        if font_data.get("foundry"):
            score += 5
        
        # Bonus for having multiple variants
        font_files = font_data.get("font_files", [])
        if len(font_files) > 1:
            score += min(len(font_files) * 2, 15)
        
        return min(score, 100)
    
    def _extract_unicode_ranges(self, font_data: Dict[str, Any]) -> List[str]:
        """Extract Unicode ranges from font data."""
        # Font Squirrel doesn't provide detailed Unicode ranges
        # Return common ranges based on classification
        classification = font_data.get("classification", {})
        classification_name = classification.get("name", "").lower()
        
        ranges = ["U+0000-00FF"]  # Basic Latin
        
        if "latin" in classification_name or "sans" in classification_name or "serif" in classification_name:
            ranges.append("U+0100-017F")  # Latin Extended
        
        return ranges
    
    def _extract_languages(self, font_data: Dict[str, Any]) -> List[str]:
        """Extract supported languages from font data."""
        # Font Squirrel doesn't provide detailed language info
        # Return common languages based on classification
        classification = font_data.get("classification", {})
        classification_name = classification.get("name", "").lower()
        
        languages = ["Latin"]
        
        if "latin" in classification_name or "sans" in classification_name or "serif" in classification_name:
            languages.append("Latin Extended")
        
        return languages
    
    def translate(self) -> Dict[str, Any]:
        """Main translation function."""
        print("Starting Font Squirrel translation...")
        
        # Fetch fonts
        raw_fonts = self.fetch_fonts()
        if not raw_fonts:
            print("No fonts fetched from Font Squirrel API")
            return self._create_empty_source()
        
        # Transform fonts
        fonts = {}
        processed = 0
        skipped = 0
        
        for font_data in raw_fonts:
            try:
                transformed = self.transform_font(font_data)
                if transformed:
                    font_id = f"squirrel.{font_data['family_name'].lower().replace(' ', '-').replace('_', '-')}"
                    fonts[font_id] = transformed
                    processed += 1
                else:
                    skipped += 1
                
                # Progress indicator
                if processed % 100 == 0:
                    print(f"Processed {processed} fonts...")
                
                # Rate limiting
                time.sleep(0.1)  # Small delay to be respectful
                
            except Exception as e:
                print(f"Error processing font: {e}")
                skipped += 1
                continue
        
        print(f"Translation complete: {processed} fonts processed, {skipped} skipped")
        
        # Create source structure
        source_data = {
            "source_info": {
                "name": "Font Squirrel",
                "description": "Free and commercial fonts from Font Squirrel",
                "url": "https://www.fontsquirrel.com",
                "api_endpoint": "https://www.fontsquirrel.com/api/fontlist/all",
                "version": "1.0",
                "last_updated": datetime.utcnow().isoformat() + "Z",
                "total_fonts": len(fonts)
            },
            "fonts": fonts
        }
        
        return source_data
    
    def _create_empty_source(self) -> Dict[str, Any]:
        """Create empty source structure for error cases."""
        return {
            "source_info": {
                "name": "Font Squirrel",
                "description": "Free and commercial fonts from Font Squirrel",
                "url": "https://www.fontsquirrel.com",
                "api_endpoint": "https://www.fontsquirrel.com/api/fontlist/all",
                "version": "1.0",
                "last_updated": datetime.utcnow().isoformat() + "Z",
                "total_fonts": 0
            },
            "fonts": {}
        }


def main():
    """Main function."""
    try:
        translator = FontSquirrelTranslator()
        source_data = translator.translate()
        
        # Write to file
        output_file = "sources/font-squirrel.json"
        os.makedirs("sources", exist_ok=True)
        
        with open(output_file, "w", encoding="utf-8") as f:
            json.dump(source_data, f, indent=2, ensure_ascii=False)
        
        print(f"Successfully generated {output_file} with {len(source_data['fonts'])} fonts")
        
    except Exception as e:
        print(f"Error: {e}")
        return 1
    
    return 0


if __name__ == "__main__":
    exit(main())

