from fastapi import FastAPI
from pydantic import BaseModel
import requests
import json

app = FastAPI()

class TextIngestRequest(BaseModel):
    app_id: str
    raw_text: str

@app.post("/parse-and-tag")
def parse_text_with_local_llm(data: TextIngestRequest):
    prompt = f"""
    Analyze the following document/text and break it into structured logical assets (e.g. Departments, Contacts, Fees, Projects, Skills).
    Extract specific phone numbers, extensions, and emails if present.

    Document Text:
    {data.raw_text}

    Return ONLY a valid JSON Array with no Markdown formatting:
    [
    {{    
        "title": "CSE Department HOD Office",
        "category": "Department_Contacts",
        "content_chunk": "Located in Block A, Room 204. Timings 9AM-5PM.",
        "contact_number": "0120-2678001 Ext: 102",
        "email_address": "cse.hod@college.edu",
        "keywords": "cse, computer science, hod, dr verma"
    }}
    ]
    """

    try:
        response = requests.post(
            "http://localhost:11434/api/generate",
            json={
                "model": "gemma2:2b",  # Ya "qwen2.5-coder:3b"
                "prompt": prompt,
                "stream": False,
                "format": "json"
            },
            timeout=45
        )
        llm_output = response.json().get("response", "[]")
        parsed_data = json.loads(llm_output)

        # Handle Dict Wrappers
        if isinstance(parsed_data, dict):
            for k in ["chunks", "data", "results", "sections", "items"]:
                if k in parsed_data and isinstance(parsed_data[k], list):
                    parsed_data = parsed_data[k]
                    break
            else:
                parsed_data = [parsed_data]

        final_chunks = []
        if isinstance(parsed_data, list):
            for item in parsed_data:
                if isinstance(item, dict):
                    # Smart key resolution (extract content regardless of model's key naming)
                    content = item.get("content_chunk") or item.get("content") or item.get("text") or item.get("chunk") or str(item)
                    tag = item.get("meta_tag") or item.get("tag") or item.get("section") or "General"

                    if content and len(str(content).strip()) > 0:
                        final_chunks.append({
                            "content_chunk": str(content).strip(),
                            "meta_tag": str(tag).strip()
                        })

        # Final Fallback if array was empty
        if not final_chunks:
            final_chunks = [{"content_chunk": data.raw_text, "meta_tag": "General"}]

    except Exception as e:
        print("Parsing Exception:", e)
        final_chunks = [{"content_chunk": data.raw_text, "meta_tag": "General"}]

    return {
        "app_id": data.app_id,
        "chunks": final_chunks
    }
