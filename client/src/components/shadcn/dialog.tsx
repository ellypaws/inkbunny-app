import React, { useState } from "react";
import { Copy } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
    Dialog,
    DialogClose,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
    DialogTrigger,
} from "@/components/ui/dialog";
// import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea"

export function ShowProcessedOutputDialog({ processedOutput }: { processedOutput: string }) {
    const [copySuccess, setCopySuccess] = useState('');

    const copyToClipboard = () => {
        navigator.clipboard.writeText(processedOutput).then(() => {
            setCopySuccess('Copied!');
            setTimeout(() => setCopySuccess(''), 2000); // Reset copy success message after 2 seconds
        }, (err) => {
            console.error('Failed to copy text: ', err);
        });
    };

    return (
        <Dialog defaultOpen={true}>
            <DialogTrigger asChild data-state={"open"}>
                <Button variant="outline">Show output</Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
                <DialogHeader>
                    <DialogTitle>Processed Output</DialogTitle>
                    <DialogDescription>
                        Below is the processed output from the server.
                    </DialogDescription>
                </DialogHeader>
                <div className="flex items-center space-x-2 ease-in-out transition-all">
                    <div className="grid flex-1 gap-2">
                        <Label htmlFor="processedOutput" className="sr-only">
                            Processed Output
                        </Label>
                        <Textarea
                            id="processedOutput"
                            defaultValue={processedOutput}
                            readOnly
                            className={"h-80 overscroll-auto max-h[250px]"}
                            placeholder="Type your message here."
                        />
                    </div>
                    <Button onClick={copyToClipboard} size="sm" className="px-3">
                        <span className="sr-only">Copy</span>
                        <Copy className="h-4 w-4" />
                    </Button>
                    {copySuccess && <p>{copySuccess}</p>}
                </div>
                <DialogFooter className="sm:justify-start">
                    <DialogClose asChild>
                        <Button type="button" variant="secondary">
                            Close
                        </Button>
                    </DialogClose>
                </DialogFooter>
            </DialogContent>
        </Dialog>
    );
}
