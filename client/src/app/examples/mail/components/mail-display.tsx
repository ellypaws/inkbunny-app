import addDays from "date-fns/addDays"
import addHours from "date-fns/addHours"
import format from "date-fns/format"
import nextSaturday from "date-fns/nextSaturday"
import {
  Archive,
  Clock,
  Forward,
  MoreVertical,
  Reply,
  ReplyAll,
  Trash2,
} from "lucide-react"

import {
  DropdownMenuContent,
  DropdownMenuItem,
} from "@/registry/default/ui/dropdown-menu"
import {
  Avatar,
  AvatarFallback,
  AvatarImage,
} from "@/registry/new-york/ui/avatar"
import { Button } from "@/registry/new-york/ui/button"
import { Calendar } from "@/registry/new-york/ui/calendar"
import {
  DropdownMenu,
  DropdownMenuTrigger,
} from "@/registry/new-york/ui/dropdown-menu"
import { Label } from "@/registry/new-york/ui/label"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/registry/new-york/ui/popover"
import { Separator } from "@/registry/new-york/ui/separator"
import { Switch } from "@/registry/new-york/ui/switch"
import { Textarea } from "@/registry/new-york/ui/textarea"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/registry/new-york/ui/tooltip"
import {FileItem, MailItem} from "@/app/examples/mail/data"

interface MailDisplayProps {
  mail: MailItem | null
}
import ReactHtmlParser from "react-html-parser";
import DOMPurify from "dompurify";
import {SkeletonCard} from "@/components/shadcn/skeleton.tsx";
import {
  Carousel,
  CarouselApi,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious
} from "@/components/ui/carousel.tsx";
import {useEffect, useState} from "react";
import {Card} from "@radix-ui/themes";
import {CardContent} from "@/components/ui/card.tsx";
import {ScrollArea} from "@/registry/new-york/ui/scroll-area";
import {ShowProcessedOutputDialog} from "@/components/shadcn/dialog.tsx";
import {MultiStepLoaderDemo} from "@/components/aceternity/loader.tsx";

interface Message {
  role: 'user' | 'system'; // Adjust the Role type according to your project's definitions
  content: string;
}

interface LLMRequest {
  messages: Message[];
  temperature: number;
  max_tokens: number;
  stream: boolean;
  // StreamChannel is omitted since it's not serializable and not relevant for the client-side request
}

interface InferenceRequest {
  config: {}; // Leave blank as instructed
  request: LLMRequest; // This will be filled with the response from /api/prefill
}

export function MailDisplay({ mail }: MailDisplayProps) {
  const [api, setApi] = useState<CarouselApi | null>(null);
  const [current, setCurrent] = useState(0);
  const [loadingImages, setLoadingImages] = useState<{ [key: number]: boolean }>({});
  const [textareaContent, setTextareaContent] = useState('');
  const [loadingPrefill, setLoadingPrefill] = useState(false);
  const [inferenceResponse, setInferenceResponse] = useState<any>(null);
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [processedOutput, setProcessedOutput] = useState("");
  const [loading, setLoading] = useState(false);
  const [llmStep, setLLMStep] = useState(0);

  const handleSendClick = async (event: React.MouseEvent<HTMLButtonElement>) => {
    event.preventDefault(); // Prevent the default form submission behavior
    setLoading(true);
    setLLMStep(0);

    if (!mail || !mail.text) return;

    // First, POST to /api/prefill
    try {
      const prefillResponse = await fetch('/api/prefill?output=complete', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ description: mail.text }),
      });
      setLLMStep(1);

      if (!prefillResponse.ok) {
        const errorData = await prefillResponse.json(); // Assuming the API returns error details in JSON
        throw new Error(errorData.message || 'Failed to prefill'); // Use the message from the API or a generic one
      }
      setLLMStep(2);

      const prefillData = await prefillResponse.json();

      console.log("prefillData", prefillData)

      setLLMStep(3);
      // Then, use prefillData to POST to /api/llm
      const llmResponse = await fetch('/api/llm?localhost=true', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ request: prefillData }),
      });

      if (!llmResponse.ok) {
        const errorData = await llmResponse.json(); // Assuming the API returns error details in JSON
        throw new Error(errorData.message || 'Failed to process with LLM'); // Use the message from the API or a generic one
      }

      setLLMStep(4);
      const llmData = await llmResponse.json();
      setProcessedOutput(JSON.stringify(llmData, null, 2));
      setIsDialogOpen(true);
      setLoading(false);

    } catch (error) {
      console.error("Error during processing:", error.message); // Log the error message
      // Optionally, update the UI to reflect the error state
    }
  };


  useEffect(() => {
    async function prefillTextarea() {
      if (mail && mail.text) {
        setLoadingPrefill(true);
        try {
          const response = await fetch(`/api/prefill?output=json`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({ description: mail.text }),
          });

          if (!response.ok) {
            throw new Error('Network response was not ok');
          }

          const data = await response.json();

          setTextareaContent(JSON.stringify(data, null, 2));
        } catch (error) {
          console.error('Error prefilling the textarea:', error);
          setTextareaContent(mail.text); // Fallback to original mail text on error
        } finally {
          setLoadingPrefill(false);
        }
      }
    }

    prefillTextarea().then(r => r);
  }, [mail]);

  useEffect(() => {
    if (mail && mail.files) {
      // Initialize all images as loading
      const initialLoadingStates = mail.files.reduce((acc, _, index) => ({
        ...acc,
        [index]: true, // Set loading state to true for each image
      }), {});
      setLoadingImages(initialLoadingStates);
    }
  }, [mail]);

  useEffect(() => {
    if (!api) return;
    setCurrent(api.selectedScrollSnap());
    api.on('select', () => {
      setCurrent(api.selectedScrollSnap());
    });
  }, [api]);


  const today = new Date()

  const myCustomPolicy = {
    ADD_TAGS: ['style'],
    ALLOWED_TAGS: ['b', 'i', 'u', 'strong', 'em', 's', 'strike', 'del', 'span', 'br', 'a', 'img', 'blockquote', 'ul', 'ol', 'li', 'dl', 'dt', 'dd', 'table', 'thead', 'tbody', 'tfoot', 'tr', 'th', 'td', 'pre', 'code', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'div', 'style'],
    ALLOWED_ATTR: ['href', 'title', 'src', 'alt', 'border', 'cellpadding', 'cellspacing', 'colspan', 'rowspan', 'style'],
    ADD_ATTR: ['style'],
  };

  DOMPurify.addHook('uponSanitizeElement', (node, data) => {
    if (node.tagName === 'STYLE' && data.tagName === 'style') {
    }
    if (data.tagName === 'img') {
        const src = node.getAttribute('src');
        if (src) {
            // Rewrite the src attribute of <img> tags
            const newSrc = `/api/image?url=${encodeURIComponent(src)}`;
            node.setAttribute('src', newSrc);
        }
    }
    if (data.tagName === 'a') {
      const href = node.getAttribute('href');
      if (href && !href.startsWith('http')) {
        node.setAttribute('href', `https://inkbunny.net${href}`);
      }
      // set style to green with broken lines hover with tailwind
        node.setAttribute('class', 'text-green-600 hover:text-green-500 hover:underline');
    }
  });

  const sanitize = (html: string) => {
    // const htmlWithPrependedLinks = html.replace(/(href=["'])(?!https?:\/\/)([^"']+)/g, '$1https://inkbunny.net/$2');
    return DOMPurify.sanitize(html, {
      WHOLE_DOCUMENT: false,
      KEEP_CONTENT: true,
      FORCE_BODY: true,
      USE_PROFILES: { html: true },
      ALLOW_DATA_ATTR: true,
      ...myCustomPolicy,
    });
  };

  const imageUrl = mail && mail.photo ? `/api/image?url=${encodeURIComponent(mail.photo)}` : '';

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center p-2">
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <Archive className="h-4 w-4" />
                <span className="sr-only">Archive</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Archive</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <Archive className="h-4 w-4" />
                <span className="sr-only">Move to junk</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Move to junk</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <Trash2 className="h-4 w-4" />
                <span className="sr-only">Move to trash</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Move to trash</TooltipContent>
          </Tooltip>
          <Separator orientation="vertical" className="mx-1 h-6" />
          <Tooltip>
            <Popover>
              <PopoverTrigger asChild>
                <TooltipTrigger asChild>
                  <Button variant="ghost" size="icon" disabled={!mail}>
                    <Clock className="h-4 w-4" />
                    <span className="sr-only">Snooze</span>
                  </Button>
                </TooltipTrigger>
              </PopoverTrigger>
              <PopoverContent className="flex w-[535px] p-0">
                <div className="flex flex-col gap-2 border-r px-2 py-4">
                  <div className="px-4 text-sm font-medium">Snooze until</div>
                  <div className="grid min-w-[250px] gap-1">
                    <Button
                      variant="ghost"
                      className="justify-start font-normal"
                    >
                      Later today{" "}
                      <span className="ml-auto text-muted-foreground">
                        {format(addHours(today, 4), "E, h:m b")}
                      </span>
                    </Button>
                    <Button
                      variant="ghost"
                      className="justify-start font-normal"
                    >
                      Tomorrow
                      <span className="ml-auto text-muted-foreground">
                        {format(addDays(today, 1), "E, h:m b")}
                      </span>
                    </Button>
                    <Button
                      variant="ghost"
                      className="justify-start font-normal"
                    >
                      This weekend
                      <span className="ml-auto text-muted-foreground">
                        {format(nextSaturday(today), "E, h:m b")}
                      </span>
                    </Button>
                    <Button
                      variant="ghost"
                      className="justify-start font-normal"
                    >
                      Next week
                      <span className="ml-auto text-muted-foreground">
                        {format(addDays(today, 7), "E, h:m b")}
                      </span>
                    </Button>
                  </div>
                </div>
                <div className="p-2">
                  <Calendar />
                </div>
              </PopoverContent>
            </Popover>
            <TooltipContent>Snooze</TooltipContent>
          </Tooltip>
        </div>
        <div className="ml-auto flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <Reply className="h-4 w-4" />
                <span className="sr-only">Reply</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Reply</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <ReplyAll className="h-4 w-4" />
                <span className="sr-only">Reply all</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Reply all</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button variant="ghost" size="icon" disabled={!mail}>
                <Forward className="h-4 w-4" />
                <span className="sr-only">Forward</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>Forward</TooltipContent>
          </Tooltip>
        </div>
        <Separator orientation="vertical" className="mx-2 h-6" />
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" disabled={!mail}>
              <MoreVertical className="h-4 w-4" />
              <span className="sr-only">More</span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem>Mark as unread</DropdownMenuItem>
            <DropdownMenuItem>Star thread</DropdownMenuItem>
            <DropdownMenuItem>Add label</DropdownMenuItem>
            <DropdownMenuItem>Mute thread</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <Separator />
      {mail ? (
        <div className="flex flex-1 flex-col">
          <div className="flex items-start p-4">
            <div className="flex items-start gap-4 text-sm">
              <Avatar>
                <AvatarImage alt={mail.name} src={imageUrl} />
                <AvatarFallback>
                  {mail.name
                    .split(" ")
                    .map((chunk) => chunk[0])
                    .join("")}
                </AvatarFallback>
              </Avatar>
              <div className="grid gap-1">
                <div className="font-semibold">{mail.name}</div>
                <div className="line-clamp-1 text-xs">{mail.subject}</div>
                <div className="line-clamp-1 text-xs">
                  <a href={mail.email}>{mail.email}</a>
                </div>
              </div>
            </div>
            {mail.date && (
              <div className="ml-auto text-xs text-muted-foreground">
                {format(new Date(mail.date), "PPpp")}
              </div>
            )}
          </div>
          <Separator />
            <ScrollArea className="h-screen">
          <div className="flex-1 whitespace-pre-wrap p-4 text-sm">
            {mail && mail.files && mail.files.length > 0 ? (
                <div className="flex-1">
                  <Carousel className="w-full max-w-xs mx-auto" setApi={setApi}>
                    <CarouselContent>
                      {mail.files.map((file: FileItem, index: number) => (
                          <CarouselItem key={index}>
                            <div className="relative">
                              <Card className="pb-4">
                                <CardContent className="flex items-center justify-center p-2 rounded-lg border bg-background shadow-lg">
                              {loadingImages[index] && <SkeletonCard />}
                              <img
                                  className="w-full h-auto"
                                  src={`/api/image?url=${encodeURIComponent(file.thumbnail_url || file.thumbnail_url_noncustom || '')}`}
                                  alt={file.file_name}
                                  style={{ display: loadingImages[index] ? 'none' : 'block' }}
                                  onLoad={() => setLoadingImages(prev => ({ ...prev, [index]: false }))}
                              />
                                </CardContent>
                              </Card>
                            </div>
                          </CarouselItem>
                      ))}
                    </CarouselContent>
                    <CarouselPrevious variant={current === 0 ? "ghost" : "secondary"} />
                    <CarouselNext variant={current === (mail.files.length - 1) ? "ghost" : "secondary"} />
                  </Carousel>
                </div>
            ) : null}
              {mail.html ? ReactHtmlParser(sanitize(mail.html)) : mail.text}
          </div>

          <Separator className="mt-auto" />
          <div className="p-4">
            <form>
              <div className="grid gap-4">
                <Textarea
                  className="p-4"
                  placeholder={`Reply ${mail.name}...`}
                  value={loadingPrefill ? 'Loading...' : textareaContent}
                  onChange={(e) => setTextareaContent(e.target.value)}
                  disabled={loadingPrefill}
                />
                <div className="flex items-center">
                  <Label
                    htmlFor="mute"
                    className="flex items-center gap-2 text-xs font-normal"
                  >
                    <Switch id="mute" aria-label="Mute thread" /> Mute this
                    thread
                  </Label>
                  <Button
                    onClick={handleSendClick}
                    size="sm"
                    className="ml-auto"
                  >
                    Send
                  </Button>

                  {isDialogOpen && (
                      <ShowProcessedOutputDialog processedOutput={processedOutput} />
                  )}
                </div>
              </div>
            </form>
          </div>
            </ScrollArea>
        </div>
      ) : (
        <div className="p-8 text-center text-muted-foreground">
          No message selected
        </div>
      )}

      <MultiStepLoaderDemo loading={loading} setLoading={setLoading} llmStep={llmStep} />
    </div>
  )
}
