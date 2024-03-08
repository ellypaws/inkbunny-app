"use client";
import React, { useState } from "react";
import { MultiStepLoader as Loader } from "@/components/ui/multi-step-loader";
import { IconSquareRoundedX } from "@tabler/icons-react";

const loadingStates = [
    {
        text: "Logging in",
    },
    {
        text: "Getting description",
    },
    {
        text: "Querying LLM",
    },
    {
        text: "Inferencing...",
    },
    {
        text: "Done",
    },
];

export function MultiStepLoaderDemo({ loading, setLoading, llmStep}: { loading: boolean ; setLoading: React.Dispatch<React.SetStateAction<boolean>>; llmStep: number }) {
    // const [loading, setLoading] = useState(false);
    if (!loading) {
        return <></>;
    }
    return (
        <div className="w-full h-[60vh] flex items-center justify-center">
            {/* Core Loader Modal */}
            <Loader loadingStates={loadingStates} loading={loading} step={llmStep} />
            {loading && (
                <button
                    className="fixed top-4 right-4 text-black dark:text-white z-[120]"
                    onClick={() => setLoading(false)}
                >
                    <IconSquareRoundedX className="h-10 w-10" />
                </button>
            )}
        </div>
    );
}
