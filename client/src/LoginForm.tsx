import {useForm} from 'react-hook-form';
import {zodResolver} from '@hookform/resolvers/zod';
import {formSchema} from '@/schema/formSchema';
import {Card, CardContent, CardDescription, CardFooter, CardHeader, CardTitle} from '@/components/ui/card';
import {Button} from "@/components/ui/button";
import {Input} from '@/components/ui/input';
import {Form, FormControl, FormField, FormItem, FormLabel, FormMessage} from '@/components/ui/form';
import {
    Alert,
    AlertDescription,
    AlertTitle,
} from "@/components/ui/alert";

import {AlertCircle, Terminal} from "lucide-react"
import {useEffect, useState} from "react";

// The rest of your LoginForm component...

export function AlertDemo({className}: {className?: string}) {
    return (
        <Alert className={`${className} fixed max-w-5xl animate-fadeIn`}>
            <Terminal className="h-4 w-4"/>
            <AlertTitle>Logged in</AlertTitle>
            <AlertDescription>
                You have successfully logged in as {localStorage.getItem('username') || 'guest'}.
            </AlertDescription>
        </Alert>
    )
}

export function AlertDestructive({className}: {className?: string}) {
    return (
        <Alert className={`${className} fixed max-w-5xl animate-fadeIn`} variant="destructive">
            <AlertCircle className="h-4 w-4"/>
            <AlertTitle>Error</AlertTitle>
            <AlertDescription>
                Your session has expired. Please log in again.
            </AlertDescription>
        </Alert>
    )
}


const LoginForm = () => {
    const form = useForm({
        resolver: zodResolver(formSchema),
        defaultValues: {
            username: '',
            password: '',
        },
    });

    const [loginStatus, setLoginStatus] = useState<null | 'success' | 'failure'>(null);
    const [showAlert, setShowAlert] = useState(false);
    const [alertAnimation, setAlertAnimation] = useState('');

    useEffect(() => {
        if (loginStatus === 'success' || loginStatus === 'failure') {
            setAlertAnimation('animate-fadeIn');
            setTimeout(() => {
                setAlertAnimation('animate-fadeOut');
            }, 3000); // Fade out after 3 seconds
        }
    }, [loginStatus]);

    const onSubmit = async (values: any) => {
        const response = await fetch('/api/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(values),
        });

        if (response.ok) {
            setLoginStatus('success');
        } else {
            setLoginStatus('failure');
        }

        // Show the alert and apply animation
        setShowAlert(true);
        setTimeout(() => {
            setShowAlert(false); // This will trigger the fade-out animation
            setTimeout(() => setLoginStatus(null), 500); // Ensure this matches the fadeOut animation duration
        }, 30000); // Keep the alert visible for 3 seconds
    };

    return (
        <div className="flex flex-col justify-between items-center">
            {showAlert && loginStatus === 'success' && (
                <div className={`${alertAnimation} fixed top-5 right-5`}>
                    <AlertDemo className="fixed top-5 right-5 animate-fadeIn"/>
                </div>
            )}
            {showAlert && loginStatus === 'failure' && (
                <div className={`${alertAnimation} fixed top-5 right-5`}>
                    <AlertDestructive className="fixed top-5 right-5 animate-fadeIn"/>
                </div>
            )}
            <Card className="w-[350px]">
                <CardHeader>
                    <CardTitle>Login</CardTitle>
                    <CardDescription>Access your account.</CardDescription>
                </CardHeader>
                <CardContent>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                            <FormField
                                control={form.control}
                                name="username"
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Username</FormLabel>
                                        <FormControl>
                                            <Input placeholder="guest" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="password"
                                render={({field}) => (
                                    <FormItem>
                                        <FormLabel>Password</FormLabel>
                                        <FormControl>
                                            <Input type="password" placeholder="Password" {...field} />
                                        </FormControl>
                                        <FormMessage/>
                                    </FormItem>
                                )}
                            />
                        </form>
                    </Form>
                </CardContent>
                <CardFooter className="flex justify-end">
                    <Button type="submit" onClick={form.handleSubmit(onSubmit)}>Login</Button>
                </CardFooter>
            </Card>
        </div>
    );
};

export default LoginForm;
